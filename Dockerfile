FROM debian:trixie-slim

ARG DEBIAN_FRONTEND=noninteractive
WORKDIR /home/data

COPY setup.sql .
ARG TRANSIT_SYSTEM=wmata
COPY ${TRANSIT_SYSTEM}/statements.sql .

RUN apt update && apt full-upgrade --yes && apt install sqlite3 --yes && \
    sqlite3 db.sqlite < setup.sql && sqlite3 db.sqlite < statements.sql

ENTRYPOINT [ "sqlite3", "db.sqlite" ]