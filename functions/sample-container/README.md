# Sample Container Function

Knative 疎通確認用の最小 Function container です。

## エンドポイント

- `GET /health`
- `POST /invoke`

`/invoke` は以下の JSON を受けます。

```json
{
  "invocation_id": "inv-1",
  "request_id": "req-1",
  "request": {
    "method": "POST",
    "path": "/invoke"
  }
}
```

## ローカル実行

```bash
cd functions/sample-container
go run .
```

確認:

```bash
curl http://127.0.0.1:8080/health

curl -X POST http://127.0.0.1:8080/invoke \
  -H 'Content-Type: application/json' \
  -d '{
    "invocation_id": "inv-1",
    "request_id": "req-1",
    "request": {
      "method": "POST",
      "path": "/invoke"
    }
  }'
```

## コンテナビルド

```bash
cd functions/sample-container
docker build -t ghcr.io/edgebase/sample-container:latest .
```

## Knative 反映

```bash
kubectl apply -f knative-service.yaml
kubectl get ksvc -n edge-functions
kubectl get revision -n edge-functions
```

Knative URL 確認:

```bash
kubectl get ksvc sample-container -n edge-functions -o jsonpath='{.status.url}'
```

Host ヘッダ付き疎通例:

```bash
SERVICE_URL=$(kubectl get ksvc sample-container -n edge-functions -o jsonpath='{.status.url}')
SERVICE_HOST=$(echo "$SERVICE_URL" | sed 's#https\?://##')
INGRESS_IP=$(kubectl get svc kourier -n kourier-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

curl -H "Host: ${SERVICE_HOST}" "http://${INGRESS_IP}/health"

curl -H "Host: ${SERVICE_HOST}" \
  -H 'Content-Type: application/json' \
  -d '{"invocation_id":"inv-1","request_id":"req-1","request":{"method":"POST","path":"/invoke"}}' \
  "http://${INGRESS_IP}/invoke"
```
