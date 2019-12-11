#!/bin/sh
TIME_STAMP=$"`date "+%Y%m%d-%H%M"`"
echo '##vso[task.setvariable variable=TIME_STAMP]'${TIME_STAMP}
