# SSL-CERT-CHECKER

检查网站的 SSL 证书情况，生成报告，并发送相关告警。

目前版本支持企业微信的 Webhook 告警。

## 使用方法

首先 Clone 源代码，进行编译：

```shell
go build .
```

然后复制 config.example.yml 文件重命名为 config.yml，进行配置：

```yaml
domains:
  - domain: www.baidu.com
    name: 百度
#   ignore_server_name: true
days_before_expire: 30
webhook_url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx"
```

其中：

- `domains` 可设置一个或多个域名。
- `ignore_server_name` 为可选项。如果设置为 true，则不校验证书中的域名和访问的域名是否一致。由于 Golang 的内部实现机制有一些 bug，如果你访问的 domain 带有非 443 端口，需要将其设置为 true。
- `webhook_url` 为企业微信的机器人 Hook 地址 URL。
- `days_before_expire` 设置报警的最小时间，如 30 则在过期前 30 天内报警。

设置完成并运行测试后，可以结合 `crontab` 等工具定期执行。
