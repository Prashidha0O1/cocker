package main 

import (
  "os"
  "os/exec"
  "fmt"
)

func main(){
  switch os.Args[1]{
  case "run":
    run()

  default:
    panic("skibidi command, try again")
  }
}

func run(){

  fmt.Println("Running: %v", os.Args[2:])

  cmd := exec.Command(os.Args[2], os.Args[3:]...)
  cmd.Stdin = os.Stdin
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: systcall.CLONE_NEWUTS,

  }
  cmd.Run()

}

