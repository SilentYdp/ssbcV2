package smart_contract

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudflare/cfssl/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/ssbcV2/meta"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

var DockerClient *client.Client
var ctx context.Context

func init() {
	ctx = context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	DockerClient = cli
}



//基于源代码编译为docker镜像
func BuildAndRun(path string,name string) {
	//超时退出设置
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	//defer cancel()

	//先压缩源代码文件
	tar, err := archive.TarWithOptions(path, &archive.TarOptions{})
	log.Info("合约编译镜像地址为：",path)
	//获取当前程序执行的路径
	file, _ := os.Getwd()
	log.Info("当前程序执行路径:", file)
	if err != nil {
		fmt.Println("tar err:",err)
	}

	//build参数
	opts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{name},
		Labels: map[string]string{
			"name": name,
		},
		Remove:     true,
	}

	res, err := DockerClient.ImageBuild(ctx, tar, opts)
	if err != nil {
		fmt.Println("ImageBuild err:", err)
	}
	defer res.Body.Close()

	err = printError(res.Body)
	if err != nil {
		fmt.Println("err:", err)
	}

	//把build成功的镜像run为容器
	exports := make(nat.PortSet, 10)
	port, err := nat.NewPort("tcp", "8080")
	if err!=nil{
		log.Error(err)
	}
	exports[port] = struct{}{}
	config := &container.Config{Image: name, ExposedPorts: exports}

	//这里嵌入一个逻辑--寻找当前空闲的端口号
	//获取到可用的tcp连接端口
	availPInt,_ := getFreePort()
	fmt.Println("可用端口号为：",availPInt)
	availPStr:=strconv.Itoa(availPInt)
	portBind := nat.PortBinding{HostPort: availPStr}
	portMap := make(nat.PortMap, 0)
	tmp := make([]nat.PortBinding, 0, 1)
	tmp = append(tmp, portBind)
	portMap[port] = tmp
	hostConfig := &container.HostConfig{PortBindings: portMap}
	// networkingConfig := &network.NetworkingConfig{}
	containerName := "contract"+name
	body, err := DockerClient.ContainerCreate(ctx, config, hostConfig, nil,nil, containerName)
	if err!=nil{
		log.Error(err)
	}
	fmt.Printf("容器build成功，ID: %s\n", body.ID)
	if err := DockerClient.ContainerStart(ctx, body.ID, types.ContainerStartOptions{}); err != nil {
		log.Error(err)
		panic(err)
	}

}

func ContractDataServer(key string) (data []byte) {
	//智能合约数据服务
	params := url.Values{}
	Url, err := url.Parse("http://docker.for.mac.host.internal:9999/query")
	if err!=nil{
		fmt.Println("url parse err:",err)
	}

	params.Set("queryKey",key)
	//如果参数中有中文参数,这个方法会进行URLEncode
	Url.RawQuery = params.Encode()
	urlPath := Url.String()

	resp,err := http.Get(urlPath)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	return body
}

//调用智能合约
func CallContract(name string,method string,args map[string]string) (retErr error,resp meta.ContractResponse) {
	//step0：先进行参数校验
	if name==""|| method==""{
		retErr=errors.New("invalid call params")
		log.Error(retErr)
		return
	}
	//step1：判断是否存在所调用的智能合约
	//先取出当前运行的容器列表
	var find bool
	cs:= ListAllContains()
	cont:=types.Container{}
	for _, c := range cs {
		fmt.Println(c.ID, c.Image)
		if c.Image==name{
			cont=c
			find=true
		}
	}
	if find==false{
		retErr=errors.New("contract not exists")
		log.Error(retErr)
		return
	}

	//解析出所调用的合约地址
	var port uint16
	for _,p:=range cont.Ports{
		port=p.PublicPort
		break
	}
	portStr:=strconv.Itoa(int(port))


	//封装调用参数
	req:=meta.ContractRequest{
		Method: method,
		Args:   args,
	}
	reqByte,_:=json.Marshal(req)
	body:=bytes.NewBuffer(reqByte)
	res,err := http.Post("127.0.0.1:"+portStr, "application/json;charset=utf-8", body)
	if err!=nil{
		log.Error("[CallContract] call err:",err)
		retErr=err
		return
	}

	//读取合约调用结果
	result, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err!=nil{
		log.Error("[CallContract] read result err:",err)
		retErr=err
		return
	}
	//反序列化为最终response
	response:=meta.ContractResponse{}
	retErr=json.Unmarshal(result,&response)
	if retErr!=nil{
		log.Error("[CallContract] json unmarshal failed,err:",err)
		return
	}
	return nil,response

}


func printError(rd io.Reader) error {
	type ErrorDetail struct {
		Message string `json:"message"`
	}

	type ErrorLine struct {
		Error       string      `json:"error"`
		ErrorDetail ErrorDetail `json:"errorDetail"`
	}

	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		fmt.Println(scanner.Text())
	}

	errLine := &ErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

//Run a container--运行一个容器
func RunAContainer() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"echo", "hello world"},
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}

//Run a container in the background
func main2() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	imageName := "bfirsh/reticulate-splines"

	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, out)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	fmt.Println(resp.ID)
}

//List and manage containers
func ListAllContains() []types.Container{
	containers, err := DockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}
	return containers
	//for _, container := range containers {
	//	fmt.Println(container.ID, container.Image)
	//}
}

//Stop all running containers
func main4() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Print("Stopping container ", container.ID[:10], "... ")
		if err := cli.ContainerStop(ctx, container.ID, nil); err != nil {
			panic(err)
		}
		fmt.Println("Success")
	}
}

//Print the logs of a specific container
func main5() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	options := types.ContainerLogsOptions{ShowStdout: true}
	// Replace this ID with a container that really exists
	out, err := cli.ContainerLogs(ctx, "f1064a8a4c82", options)
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, out)
}

//List all images
func main6() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		panic(err)
	}

	for _, image := range images {
		fmt.Println(image.ID)
	}
}

//Pull an image
func main7() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	out, err := cli.ImagePull(ctx, "alpine", types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	defer out.Close()

	io.Copy(os.Stdout, out)
}

//Commit a container--更新容器commit为镜像
func main8() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	createResp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"touch", "/helloworld"},
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, createResp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	commitResp, err := cli.ContainerCommit(ctx, createResp.ID, types.ContainerCommitOptions{Reference: "helloworld"})
	if err != nil {
		panic(err)
	}

	fmt.Println(commitResp.ID)
}

//在本机找空闲端口号
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
