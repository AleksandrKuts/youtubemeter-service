#!/bin/bash

DIR_SERVICE="/home/youtubemetric/collect"


cd $DIR_SERVICE

CMD=$DIR_SERVICE/collector
CURRENT_PID="$CMD  -config ./collector.ini"

S=`pgrep -f $CMD`

if [[ -z $S ]]
then
    $CURRENT_PID >> $DIR_SERVICE/log/collector.log 2>&1 &
else
    echo "Process already running"
fi