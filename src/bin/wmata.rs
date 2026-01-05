use anyhow::{Context, Error, Result};
use fire_alarm::Incident;
use fire_alarm::Parser;
use reqwest::Url;
use reqwest::header::HeaderValue;
use serde::Deserialize;

#[derive(Parser)]
#[command(version)]
struct Cli {
    /// API key for fetching incidents from WMATA
    #[arg(short, long)]
    key: HeaderValue,

    #[arg(short, long, default_value_t = Url::parse("https://api.wmata.com/Incidents.svc/json/Incidents").unwrap())]
    endpoint: Url,

    #[command(flatten)]
    args: fire_alarm::Args,

    #[cfg(feature = "log")]
    #[command(flatten)]
    verbosity: clap_verbosity_flag::Verbosity,
}

#[tokio::main]
async fn main() {
    #[cfg(feature = "argfile")]
    let args = Cli::parse_from(
        argfile::expand_args(argfile::parse_fromfile, argfile::PREFIX)
            .expect("Failed to read from ArgFile"),
    );

    #[cfg(not(feature = "argfile"))]
    let args = Cli::parse();

    let incidents = fetch_incidents(args.endpoint, args.key);

    #[cfg(feature = "log")]
    env_logger::Builder::new().filter_level(args.verbosity.log_level_filter()).init();

    let incidents: Vec<Incident> = incidents
        .await
        .expect("Failed to fetch incidents from WMATA")
        .try_into()
        .expect("Failed to Convert to FireAlarm Incidents");

    #[cfg(feature = "log")]
    log::debug!("{incidents:?}");

    let args = args.args;
    fire_alarm::run(
        args.username,
        args.password,
        &args.relay,
        args.address,
        incidents,
        args.timestamp,
        args.database,
    )
    .await
    .expect("Failed to run FireAlarm");
}

#[allow(non_snake_case)]
#[derive(Deserialize)]
struct IncidentsWmata {
    Incidents: Vec<IncidentWmata>, // Array containing rail disruption information
}

#[allow(non_snake_case, dead_code)]
#[derive(Deserialize)]
struct IncidentWmata {
    DateUpdated: String, // Date and time (Eastern Standard Time) of last update. Will be in YYYY-MM-DDTHH:mm:SS format (e.g.: 2010-07-29T14:21:28).
    // DelaySeverity: String, // Deprecated
    Description: String, // Free-text description of the incident.
    // EmergencyText: String, // Deprecated
    // EndLocationFullName: String, // Deprecated
    IncidentID: Option<String>,   // Unique identifier for an incident.
    IncidentType: Option<String>, // Free-text description of the incident type. Usually Delay or Alert but is subject to change at any time.
    LinesAffected: Option<String>, // Semi-colon and space separated list of line codes (e.g.: RD; or BL; OR; or BL; OR; RD;).
                                   // PassengerDelay: String, // Deprecated
                                   // StartLocationFullName: String, // Deprecated
}

async fn fetch_incidents(endpoint: Url, key: HeaderValue) -> Result<IncidentsWmata> {
    Ok(reqwest::Client::builder()
        .build()
        .map_err(Error::from)?
        .get(endpoint)
        .header("api_key", key)
        .send()
        .await
        .map_err(Error::from)?
        .json::<IncidentsWmata>()
        .await?)
}

impl TryInto<Vec<Incident>> for IncidentsWmata {
    type Error = Error;
    fn try_into(self) -> Result<Vec<Incident>> {
        self.Incidents
            .into_iter()
            .map(|incident| incident.try_into()?)
            .collect()
    }
}

impl TryInto<Incident> for IncidentWmata {
    type Error = Error;
    fn try_into(self) -> Result<Incident> {
        use chrono::TimeZone;
        let eastern_datetime = chrono::NaiveDateTime::parse_from_str(&self.DateUpdated, "%FT%T")?;
        let eastern_datetime_tz = chrono_tz::US::Eastern
            .from_local_datetime(&eastern_datetime)
            .single()
            .context(
                "Parsed datetime falls in a fold or gap in US Eastern timezone or there was an error.",
            )?;
        Ok(Incident::new(
            eastern_datetime_tz.to_utc(),
            self.Description,
        ))
    }
}

impl Into<Result<Incident>> for IncidentWmata {
    fn into(self) -> Result<Incident> {
        self.try_into()
    }
}

#[cfg(test)]
mod test {
    use std::env;

    use super::fetch_incidents;

    #[tokio::test]
    async fn validate_api_key() {
        let url = reqwest::Url::parse("https://api.wmata.com/Misc/Validate").unwrap();
        let key = env::var("API_KEY").unwrap();
        let success = reqwest::Client::builder()
            .build()
            .unwrap()
            .get(url)
            .header("api_key", key)
            .send()
            .await
            .unwrap()
            .status()
            .is_success();
        assert!(success);
    }

    #[tokio::test]
    async fn test_fetch_incidents() {
        let endpoint = env::var("ENDPOINT")
            .unwrap_or_else(|_| String::from(
                "https://api.wmata.com/Incidents.svc/json/Incidents",
            ))
            .parse()
            .unwrap();
        let key = env::var("API_KEY").unwrap().try_into().unwrap();
        fetch_incidents(endpoint, key).await.unwrap();
    }

    #[ignore]
    #[tokio::test]
    async fn test_main() {
        let endpoint = env::var("ENDPOINT")
            .unwrap_or_else(|_| String::from(
                "https://api.wmata.com/Incidents.svc/json/Incidents",
            ))
            .parse()
            .unwrap();
        let key = env::var("API_KEY").unwrap().try_into().unwrap();
        let incidents = fetch_incidents(endpoint, key);

        let timestamp = env::var("TIMESTAMP").unwrap_or_else(|_| String::from("timestamp.txt"));
        let database = env::var("DATABASE").unwrap_or_else(|_| String::from("sqlite::memory:"));
        let address = lettre::Address::new("obi.wan", "konobi.com").unwrap();

        let incidents: Vec<fire_alarm::Incident> = incidents.await.unwrap().try_into().unwrap();
        fire_alarm::test_run(incidents, timestamp, database.into(), address)
            .await
            .expect("Failed to run FireAlarm");
    }
}
