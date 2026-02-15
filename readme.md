# Server Restart programm

## Send command from the programm

At present all scripts are located at server, so it's needed to run only 1 command from here. But if it will be needed to run script from here, there're some thoughts.

All commands in script can be divided by 2 groups:

1. Command that must just run once, no result expected
2. Command that must run once or multiple times and after that the output must be empty or must not be empty

So there's config example for scripts part for server with such idiom:

```yaml
scripts:
  - name: restart
    script:
    - command: /t24/T24/bnk/UD/jboss.sh restart &   # many commands to run at once
      repeat: 0                                     # if = 0 then 'mustBeEmpty' will be ignored else this command will run as often as 'repeat' and the result will be checked by 'mustBeEmpty'
      mustBeEmpty: false                            # if "true" result of 'command' must be empty otherwise the result must be not empty
      message: Restarted                            # message to be send on successfull result of 'command' (if 'repeat' = 0 => on any result); if 'message' is missing then message won't be send
    - command: DBTools -u $tafjUser -p $tafjPassword -s OFS "TSA.SERVICE,/I//0/0,"$OFSUser/$OFSPassword",TSM,SERVICE.CONTROL=STOP" | grep "SERVICE.CONTROL"
      repeat: 0
```

## ToDo

* limits for different handlers (only start, not NOW...)
  * **this is achieved through adding to `projects.yaml` `type` and `actions` of in&out handlers**
* reply to message for in&out handlers
* preserve chat id for in&out handlers (inside db ?)
* check actions before adding to scheduler
