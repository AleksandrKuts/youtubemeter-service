#!/bin/bash

#sudo -u postgres psql -d youtube_statistics -c "COPY playlist (idch, enable, title, timeadd, countvideo) TO '/tmp/playlist.dt';"
sudo -u postgres psql -d youtube_statistics2 -c "COPY channel FROM '/tmp/playlist.dt';"

#sudo -u postgres psql -d youtube_statistics -c "COPY video (id, chid, title, description, publishedat) TO '/tmp/video.dt';"
sudo -u postgres psql -d youtube_statistics2 -c "ALTER TABLE video DISABLE TRIGGER tr_change_video; COPY video (id, idch, title, description, publishedat) FROM '/tmp/video.dt'; ALTER TABLE video ENABLE TRIGGER tr_change_video;"

#sudo -u postgres pg_dump -t metric --data-only youtube_statistics -f /tmp/metric.dt
sudo -u postgres psql -d youtube_statistics2 -1 -f '/tmp/metric.dt';


