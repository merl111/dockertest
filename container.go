package dockertest

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// A Container is a container inside docker
type Container struct {
	Image string
	Name  string
	Args  []string
	Addr  string
	cmd   *exec.Cmd
}

func removeContainer(name string) error {
	argsFull := append([]string{"rm", "--force"}, name)
	cmd := exec.Command("docker", argsFull...)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("could not remove container, %s", err)
	}

	return nil
}

// Shutdown ends the container
func (c *Container) Shutdown() {
	c.cmd.Process.Signal(syscall.SIGINT)
	c.cmd.Process.Signal(syscall.SIGTERM)
	//  Wait till the process exits.
	c.cmd.Wait()
	removeContainer(c.Name)
}

// RunContainer runs a given docker container and returns a port on which the
// container can be reached
func RunContainer(container string, port string, name string, waitFunc func(addr string) error, args ...string) (*Container, error) {
	removeContainer(name)
	free := freePort()
	host := getHost()
	addr := fmt.Sprintf("%s:%d", host, free)
	argsFull := append([]string{"run"}, args...)
	argsFull = append(argsFull, fmt.Sprintf("%s%s", "--name=", name))
	argsFull = append(argsFull, "-e", "POSTGRES_PASSWORD=postgres")
	argsFull = append(argsFull, "-p", fmt.Sprintf("%d:%s", free, port), container)
	cmd := exec.Command("docker", argsFull...)

	// run this in the background
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("could not run container, %s", err)
	}
	for {
		err := waitFunc(addr)
		if err == nil {
			break
		}

		time.Sleep(time.Millisecond * 150)
	}

	return &Container{
		Image: container,
		Name:  name,
		Addr:  addr,
		Args:  args,
		cmd:   cmd,
	}, nil
}

func getHost() string {
	out, err := exec.Command("docker-machine", "ip", os.Getenv("DOCKER_MACHINE_NAME")).Output()
	if err == nil {
		return strings.TrimSpace(string(out[:]))
	}
	return "localhost"
}

func freePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}
