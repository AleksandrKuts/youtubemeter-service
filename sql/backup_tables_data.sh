#!/bin/bash

sudo -u postgres pg_dump --schema-only --no-privileges --no-owner --no-tablespaces youtube_statistics2 | sed -e '/^--/d' > schema.sql
sudo -u postgres pg_dump -t video --data-only youtube_statistics2 > video.sql
sudo -u postgres pg_dump -t channel --data-only youtube_statistics2 > channel.sql
sudo -u postgres pg_dump -t metric --data-only youtube_statistics2 > metric.sql
