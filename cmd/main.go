/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
  "github.com/cloudflare/cfssl/log"
  "os"
)

var (
  blockingStart = true
)

func main() {
  if err := RunMain(os.Args); err != nil {
    os.Exit(1)
  }


}
func RunMain(args []string) error {
  //主节点log日志重定向
  //os.Stderr = f
  if os.Args[3]=="8000"{
    if isExist("log"){
      os.Remove("log")
    }
    f, _ := os.OpenFile("log", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND,0755)
    os.Stdout = f
    defer f.Close()
  }



  // Save the os.Args
  saveOsArgs := os.Args
  os.Args = args

  cmdName := ""
  if len(args) > 1 {
    cmdName = args[1]
  }
  scmd := NewCommand(cmdName, blockingStart)

  // Execute the command
  err := scmd.Execute()

  // Restore original os.Args
  os.Args = saveOsArgs

  return err
}

//判断文件或文件夹是否存在
func isExist(path string) bool {
  _, err := os.Stat(path)
  if err != nil {
    if os.IsExist(err) {
      return true
    }
    if os.IsNotExist(err) {
      return false
    }
    log.Info(err)
    return false
  }
  return true
}
