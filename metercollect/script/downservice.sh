#!/bin/bash

ps aux | grep -i metercollect | grep -v grep | awk {'print $2'} | xargs kill -2

