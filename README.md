# bifurcate
This is a program that runs multiple configured programs and forwards the signals along. This could make it easier to run something like consul-template and your configured program in the same docker container. It also forwards all of the stdout/stderr from the child programs

* * *
### Why?
There are many people that believe that only 1 process should run in a conatiner at a time. I think that this makes a ton of since, because you want to monitor that the container is running not that everything inside the container is running well. Where this starts to make a little less sense is for something like [gunicorn](http://gunicorn.org/) which does really well at managing all of its sub processes. I think that it makes a lot of sense to allow something like gunicorn to run inside a container. This got me thinking that as long as the underlying processes are being monitored then it is fine.

There are many alternatives such as [supervisord](http://supervisord.org/), [systemd](http://www.freedesktop.org/wiki/Software/systemd/), etc.. but I was looking for something very light weight, and that would not restart the process. I did not want anything inside the docker container to get restarted because usually there is some kind of orchestartion around the conatiner itself and this could interfere with that. 

`bifurcate` is attempting to pretend that it is just one process so kills everything if any process dies, and will forward all signals to the underlying processes.

* * *
### Configuration
Configuration is done in [json](http://json.org/). This is easy to have structure, and is natively supported by go, so does not require adding any more dependencies.

Right now everything exists under the "programs" key which is just a map of name of program to arguments describing programs.
```json
{
 "programs":{}
}
```

To add a program to be executed just add it to that object. Such as `sleep 10`. 
###### `args`
```json
{
 "programs": {
   "sleeper": {
     "args": ["/bin/sleep", "10"]
    }
  }
}
```
The first element int the list of arguments is expected to be the executable, it is best for this to be a full path, but it does not have to be. All of the environmental variables that are given to `bifurcate` when it runs are passed to all of the child programs.

Just keep adding more uniquely named programs to run as needed.

Sometimes you want programs to run in a specific order, maybe one is generating the configuration file for another. Having to specify order can sometimes lead to bugs that are difficult to find. It could be that the race only occurs in the wrong order in production. Instead of specifying order of execution, `bifurcate` allows you to specify a specific file to exist before running a program.
###### `requires`
```json
{
  "programs": {
    "consul-template": { 
     "args": ["consul-template", "-template", "/config.conf.ctmpl:/config.conf"] 
    },
    "myapp": {
      "requires": [{"file": "/config.cong"}],
      "args": ["myapp", "-config", "/config.conf"]
    }
  }
}
```
In this example we are running [`consul-template`](https://github.com/hashicorp/consul-template) to get and keep a configuration file up to date for our application. We do not want the application to start up until the configuration file exists so we tell the program that it `requires` a `file` at a specified location to exist. This way even if consul template starts up well, we waiting until it successfully writes out the file to start our application.

* * *
### Running the example.
```
BABY_NAME=quinn go run bifurcate.go resources/demo.json
```
`touch` baby-sleep file in the working directory to put the baby to sleep, and it will wake up after a while `exit 1` and kill the entire program.
