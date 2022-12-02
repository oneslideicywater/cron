## go cron

cron parse the cron syntax and execute cmd periodically

for example:

```bash
go run main.go -d 10 "echo hello > %date:~0,4%-%date:~5,2%-%date:~8,2%.log"
```