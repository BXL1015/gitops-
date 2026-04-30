# 自动化验证脚本
Write-Host "--- 验证基线环境 (env0) ---"
curl.exe "http://localhost:9001/?n=10"

Write-Host "
--- 验证泳道隔离 (env8) ---"
curl.exe -H "X-Tenant: alice" "http://localhost:9001/?n=10"
