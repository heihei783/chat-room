package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Config SMTP邮件服务器配置
type Config struct {
	Host     string // SMTP服务器地址，如 smtp.qq.com
	Port     int    // SMTP端口：465为SSL，587为STARTTLS
	Username string // 登录账号
	Password string // 密码或授权码
	From     string // 发件人邮箱地址
	SSL      bool   // 是否使用SSL（true=465隐式TLS，false=587 STARTTLS）
}

// Sender 基于标准库 net/smtp 的邮件发送器，无需第三方依赖
type Sender struct {
	cfg Config
}

// NewSender 创建邮件发送器
func NewSender(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

// Send 发送一封邮件：to为收件人邮箱，subject为标题，body为正文
func (s *Sender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	// 1. 建立TCP连接；SSL模式下（465端口）直接进行TLS握手
	var conn net.Conn
	var err error
	if s.cfg.SSL {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.Host})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("连接SMTP服务器失败: %w", err)
	}
	defer conn.Close()

	// 2. 创建SMTP客户端
	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %w", err)
	}
	defer client.Close()

	// 3. 非SSL模式（587端口）通过STARTTLS升级为加密连接
	if !s.cfg.SSL {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return fmt.Errorf("STARTTLS升级失败: %w", err)
		}
	}

	// 4. 使用账号密码进行SMTP认证
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP认证失败: %w", err)
	}

	// 5. 设置发件人与收件人
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}

	// 6. 写入邮件内容（含MIME头，支持中文与UTF-8编码）
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("开始写入邮件内容失败: %w", err)
	}
	if _, err := w.Write([]byte(buildMessage(s.cfg.From, to, subject, body))); err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("结束写入邮件内容失败: %w", err)
	}

	// 7. 发送并关闭连接
	return client.Quit()
}

// buildMessage 组装带MIME头的邮件内容
func buildMessage(from, to, subject, body string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}
