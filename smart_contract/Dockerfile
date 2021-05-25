# 拉取 Go 语言最新的基础镜像
FROM golang:latest

# 在容器内设置 /app 为当前工作目录
WORKDIR /app

# 把文件复制到当前工作目录
COPY . .

# 设置 GOPROXY 环境变量
ENV GOPROXY="https://goproxy.cn"

# 下载全部依赖项
RUN go mod download

# 编译项目
RUN go build -o main .

# 暴露 8080 端口
EXPOSE 8080

# 执行可执行文件
CMD ["./main"]
