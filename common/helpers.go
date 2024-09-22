package common

import "os/user"

func IsRunningAsRoot() bool {
    usr, _ := user.Current()
    return usr.Username == "root"
}
