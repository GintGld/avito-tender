#!/bin/bash

# apply migrations
./migrator -postgresURL ${POSTGRES_CONN} -migrations migrations

# entrypoint
./tender