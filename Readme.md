# Arg
```
go run ./arg 2 
```


# Automation
crontab -e

1분마다 실행(테스트용)
* * * * * cd /Users/ysunny/Workspace/cmsn-bot-rotate && bash -c 'source .env.sh && make run' >> /Users/ysunny/Workspace/cmsn-bot-rotate/make_run.log 2>&1

3시간마다 실행
CMSN_ROTATE_PATH = 
0 */3 * * * cd ${CMSN_ROTATE_PATH} && bash -c 'source .env.sh && make run' >> ${CMSN_ROTATE_PATH}/make_run.log 2>&1