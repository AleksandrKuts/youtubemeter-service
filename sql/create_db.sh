#!/bin/bash

sudo -u postgres createuser -P youtubemetric
sudo -u postgres createdb -O youtubemetric -E Unicode -T template0 youtube_statistics
cat schema.sql | sudo -u postgres psql -U youtubemetric youtube_statistics
