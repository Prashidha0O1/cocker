This readme will serve as a notebook for my notes while I am making a simple container like Docker.

### How Namespaces Work

Namespaces limit what a process can see. When we run a container in Docker, we only see a few processes running on the host. This is because Docker assigns a separate namespace for process IDs (PID namespace), preventing the containerized process from viewing all the host’s processes. Similarly, a namespace for the hostname ensures the container has its hostname, independent of the host machine.

### Setting Up Namespaces

To create these namespaces, we use syscalls. These syscalls restrict the view a process has of the host machine, making it unaware of other processes and system resources.

### What Are We Trying to Build?

The goal is to mimic Docker’s behavior using Golang. The Docker command to run a container looks like this:
```
docker run image <cmd> <args>
```
Similarly, in my Go program, we can achieve the same functionality using:
```
go run main.go run image <cmd> <args>
```

This means:
` go run main.go ` compiles and runs my executable, which is similar to executing Docker.

run is a subcommand that creates an isolated container process.

The container executes the given command with specified arguments in a namespace-restricted environment.

### Containerizing Commands Using Namespaces

The next step is to containerize the command execution by creating namespaces. In Go, we can do this using SysProcAttr within the syscall package. This allows us to structure our container’s execution environment.

Using SysProcAttr and Clone Flags

The SysProcAttr struct allows us to pass Cloneflags, which are essential for process cloning and namespace creation. This ensures the commands we run execute in a fully isolated environment.

For example:

- UTS Namespace (CLONE_NEWUTS): Isolates the hostname.

- PID Namespace (CLONE_NEWPID): Ensures the container has its process hierarchy.

Running the Program Inside Its Own Namespace

To run the program inside its namespace, we use:
```
os.Executable()
```
Instead of executing the run directly, we invoke the program using `/proc/self/exe`. This approach ensures that:
`
run reinvokes the process inside a new namespace.

child executes within the existing namespace with an isolated hostname.
`

### Execution Flow

User runs:
```
go run main.go run image <cmd> <args>
```

The program creates a new process with namespaces using Cloneflags.

The process runs the specified command inside an isolated environment.

If the child is executed, it runs with only the hostname isolated.

This approach closely mimics how Docker manages containerized processes using Linux namespaces and syscalls.
