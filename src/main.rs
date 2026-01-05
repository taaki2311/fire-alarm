use std::io;

use clap::Parser;
#[cfg(not(feature = "log"))]
use fire_alarm::Args as Cli;

#[tokio::main]
async fn main() {
    #[cfg(feature = "argfile")]
    let args = Cli::parse_from(
        argfile::expand_args(argfile::parse_fromfile, argfile::PREFIX)
            .expect("Failed to read from ArgFile"),
    );

    #[cfg(not(feature = "argfile"))]
    let args = Cli::parse();

    #[cfg(feature = "log")]
    let args = {
        env_logger::Builder::new()
            .filter_level(args.verbosity.log_level_filter())
            .init();
        args.args
    };

    let incidents: Vec<_> = serde_json::from_reader(io::BufReader::new(io::stdin()))
        .expect("Failed to parse incidents");

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

#[cfg(feature = "log")]
#[derive(Parser)]
struct Cli {
    #[command(flatten)]
    args: fire_alarm::Args,

    #[command(flatten)]
    verbosity: clap_verbosity_flag::Verbosity,
}

#[cfg(test)]
mod test {
    use std::env;

    use tokio::io;

    async fn fetch_incidents(
        path: impl AsRef<std::path::Path>,
    ) -> io::Result<Vec<fire_alarm::Incident>> {
        use io::AsyncReadExt;

        let file = tokio::fs::File::open(path);
        let mut dst = String::new();
        file.await?.read_to_string(&mut dst).await?;
        Ok(serde_json::from_str(&dst)?)
    }

    #[tokio::test]
    async fn test_fetch_incidents() {
        let path = env::var("INCIDENTS").unwrap_or_else(|_| String::from("incidents.json"));
        fetch_incidents(path).await.unwrap();
    }

    #[tokio::test]
    async fn test_connection() {
        let username = env::var("USERNAME").unwrap();
        let password = env::var("PASSWORD").unwrap();
        let relay = env::var("RELAY").unwrap();
        assert!(
            fire_alarm::test_connection(username, password, &relay)
                .await
                .unwrap()
        )
    }

    #[ignore]
    #[tokio::test]
    async fn test_main() {
        let path = env::var("INCIDENTS").unwrap_or_else(|_| String::from("incidents.json"));
        let incidents = fetch_incidents(path).await.unwrap();

        let timestamp = env::var("TIMESTAMP").unwrap_or_else(|_| String::from("timestamp.txt"));
        let database = env::var("DATABASE").unwrap_or_else(|_| String::from("sqlite::memory:"));
        let address = lettre::Address::new("obi.wan", "konobi.com").unwrap();

        fire_alarm::test_run(incidents, timestamp, database.into(), address)
            .await
            .unwrap();
    }
}
