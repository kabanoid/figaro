#!/bin/bash
echo connection URL: postgres://figaro:secret@localhost:5432/figaro
docker run --rm --name figaro-db -p 5432:5432 -e POSTGRES_USER=figaro -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=figaro postgres:9.5
