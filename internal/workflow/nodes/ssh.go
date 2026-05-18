package nodes

import (
	"RCAgent/internal/models"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHExecutor SSH执行器
type SSHExecutor struct {
	DefaultNodeExecutor
}

// NewSSHExecutor 创建SSH执行器
func NewSSHExecutor() *SSHExecutor {
	return &SSHExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "SSH",
			Category:    "connection",
			Description: "Execute commands via SSH connection",
		},
	}
}

// Execute 执行SSH命令
func (e *SSHExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	return ExecuteSSH(ctx, nodeCtx)
}

// ExecuteSSH 执行SSH命令
func ExecuteSSH(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	fmt.Println("nc.Config ->", nc.Config)
	host := resolveEnvVar(getStr(nc.Config, "host", ""))
	portStr := getStr(nc.Config, "port", "22")
	port := parsePortEnvVar(portStr)
	command := getStr(nc.Config, "command", "")
	timeout := getInt(nc.Config, "timeout", 30)

	// 获取认证信息（从SSHAuth数据源）
	authType := getStr(nc.Config, "authType", "")
	username := getStr(nc.Config, "authUsername", "")
	password := getStr(nc.Config, "authPassword", "")
	privateKey := getStr(nc.Config, "authPrivateKey", "")

	// 验证必需配置
	if host == "" {
		return errorResult("host is required - please configure in the node"), nil
	}
	if port == 0 {
		port = 22
	}
	if username == "" {
		return errorResult("username is required - please select a SSHAuth datasource"), nil
	}
	if command == "" {
		return errorResult("command is required - please enter a command to execute"), nil
	}

	log.Printf("[SSH] Connecting to %s@%s:%d", username, host, port)

	// 创建SSH客户端配置
	var authMethods []ssh.AuthMethod

	switch authType {
	case "ssh_password":
		if password != "" {
			authMethods = append(authMethods,
				ssh.Password(password),
				ssh.KeyboardInteractive(
					func(user, instruction string, questions []string, echos []bool) ([]string, error) {
						answers := make([]string, len(questions))
						for i := range answers {
							answers[i] = password
						}
						return answers, nil
					},
				),
			)
			log.Printf("[SSH] Using password authentication (with keyboard-interactive fallback)")
		} else {
			return errorResult("password is required for ssh_password auth method"), nil
		}
	case "ssh_key":
		if privateKey != "" {
			signer, err := ssh.ParsePrivateKey([]byte(privateKey))
			if err != nil {
				return errorResult(fmt.Sprintf("failed to parse private key: %v", err)), nil
			}
			authMethods = append(authMethods, ssh.PublicKeys(signer))
			log.Printf("[SSH] Using private key authentication")
		} else {
			return errorResult("privateKey is required for ssh_key auth method"), nil
		}
	default:
		// 尝试密码认证 (password + keyboard-interactive)
		if password != "" {
			authMethods = append(authMethods,
				ssh.Password(password),
				ssh.KeyboardInteractive(
					func(user, instruction string, questions []string, echos []bool) ([]string, error) {
						answers := make([]string, len(questions))
						for i := range answers {
							answers[i] = password
						}
						return answers, nil
					},
				),
			)
		}
		// 尝试私钥认证
		if privateKey != "" {
			signer, err := ssh.ParsePrivateKey([]byte(privateKey))
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
		if len(authMethods) == 0 {
			return errorResult("no authentication method available - please configure password or privateKey in SSHAuth datasource"), nil
		}
	}

	config := &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(timeout) * time.Second,
	}

	// 连接SSH服务器
	addr := fmt.Sprintf("%s:%d", host, port)
	sshCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	conn, err := dialSSH(sshCtx, "tcp", addr, config)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to connect to SSH server: %v", err)), nil
	}
	defer conn.Close()

	// 创建session
	session, err := conn.NewSession()
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create SSH session: %v", err)), nil
	}
	defer session.Close()

	// 设置输出
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// 执行命令
	log.Printf("[SSH] Executing command: %s", command)
	if err := session.Run(command); err != nil {
		// 如果stderr有内容，返回stderr
		if stderr.Len() > 0 {
			return errorResult(fmt.Sprintf("command failed: %s, stderr: %s", err.Error(), stderr.String())), nil
		}
		return errorResult(fmt.Sprintf("command failed: %v", err)), nil
	}

	output := stdout.String()
	log.Printf("[SSH] Command executed successfully, output length: %d", len(output))

	return newResult(map[string]any{
		"output": output,
	}), nil
}

// dialSSH 带有重试的SSH连接
func dialSSH(ctx context.Context, network, address string, config *ssh.ClientConfig) (*ssh.Client, error) {
	const maxRetries = 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 先测试端口连通性
		dialer := net.Dialer{Timeout: config.Timeout}
		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			lastErr = fmt.Errorf("TCP connection failed: %w", err)
			log.Printf("[SSH] Connection attempt %d/%d failed: %v", i+1, maxRetries, err)
			continue
		}

		log.Printf("[SSH] TCP connection established, performing SSH handshake...")

		// 升级到SSH连接
		sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
		if err != nil {
			conn.Close()
			lastErr = fmt.Errorf("SSH handshake failed: %w", err)
			log.Printf("[SSH] SSH handshake attempt %d/%d failed: %v", i+1, maxRetries, err)
			continue
		}

		log.Printf("[SSH] SSH handshake successful!")
		return ssh.NewClient(sshConn, chans, reqs), nil
	}

	return nil, lastErr
}

// resolveEnvVar 解析环境变量，支持 ${VAR_NAME} 或 $VAR_NAME 格式
func resolveEnvVar(value string) string {
	if value == "" {
		return value
	}

	// 匹配 ${VAR_NAME} 格式
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		varName := match[2 : len(match)-1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match // 如果环境变量不存在，保留原样
	})
}

// parsePortEnvVar 解析端口环境变量
func parsePortEnvVar(value string) int {
	value = resolveEnvVar(value)
	// 尝试解析为数字
	var port int
	if _, err := fmt.Sscanf(value, "%d", &port); err == nil && port > 0 && port <= 65535 {
		return port
	}
	return 0
}
