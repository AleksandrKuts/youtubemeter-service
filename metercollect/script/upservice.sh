#!/bin/bash

DIR_SERVICE="/home/KR110666KAI/collect"


cd $DIR_SERVICE

CMD=$DIR_SERVICE/metercollect
CURRENT_PID="$CMD  -config ./metercollect.ini"

S=`pgrep -f $CMD`

if [[ -z $S ]]
then
    $CURRENT_PID >> $DIR_SERVICE/log/metercollect.log 2>&1 &
else
    echo "Process already running"
fi