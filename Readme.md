this readme will serve as a notebook for my notes while i am making a simple container like docker

namespaces, 
it is where we limit what process can see, when we run a container in docker
we can only see few of the process running on the host and that's because its got
a namespace for process IDs (PID for short) as it can only see its host name cause thats
because of namespacing

How to setup these namespaces ? 
using syscalls as this is a main part what it makesa container into a container. Cause restricting the view of a process that a process has the things that are goin on the host machine

So what are we trying to build????


here when you're tryin to run docker the command goes like this 
`docker run image <cmd> <args>/<params>`

this container gonna be something like this in my go program i can do similarly like docker command:

go run <filename> in my case main.go 
`go run main.go run image <cmd> <args>`

go run main.go compiles and run my main executable this is kind of a equivalent of docker

TLDR: `docker` becomes -> `go run main.go` 

then this alternate `go run main.go` to run some commands <cmd> and can be some parameters. 

