# 启动 15 个服务进行本地验证 (env0)
for ($i=1; $i -le 15; $i++) {
    $name = "svc$i"
    $port = 9000 + $i
    Write-Host "Starting $name on port $port..."
    Start-Job -ScriptBlock { 
        $name = $args[0]
        $port = $args[1]
        $env:SERVICE_NAME = $name
        $env:LISTEN_ADDR = ":$port"
        $env:SERVICE_ENV = "0"
        ./bin/$name
    } -ArgumentList $name, $port
}
Write-Host "All services started. Entrance is svc1 on http://localhost:9001"
