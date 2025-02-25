package main 

import (
  "os"
  "os/exec"
  "fmt"
  "syscall"
)

func main(){
  switch os.Args[1]{
  case "run":
    run()

  case "child":
    child()
  default:
    panic("skibidi command, try again")
  }
}

func run(){

  fmt.Println("Running: %v", os.Args[2:])

  cmd := exec.Command("/prco/self/exe", append([]string{"child"}, os.Args[2]...)...)
  cmd.Stdin = os.Stdin
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWUTS,

  }
  
  syscall.Sethostname([]byte("skibidicontainer"))
  cmd.Run()
}


func child(){
  

}
