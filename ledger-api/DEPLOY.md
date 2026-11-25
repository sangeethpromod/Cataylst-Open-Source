# Deploying ledger-api to AWS Lambda

Lambda has no "Go" runtime to pick. Go is compiled ahead of time, so you upload a
binary named `bootstrap` and run it on the **Amazon Linux 2023 custom runtime**
(`provided.al2023`). The code detects Lambda via `AWS_LAMBDA_FUNCTION_NAME` and
switches from an HTTP server to the Lambda handler automatically.

## 1. Atlas (once)

- Network Access → add `0.0.0.0/0` (Lambda IPs change; the DB user password and
  `X-API-Key` are the protection). Do **not** put the Lambda in a VPC.
- Use a database user limited to the `ledger` database (readWrite).

## 2. Create the function in the console (once)

1. AWS Console → Lambda → **Create function** → *Author from scratch*.
2. Function name: `ledger-api`.
3. Runtime: **Amazon Linux 2023** (under "Custom runtime on Amazon Linux 2023",
   id `provided.al2023`).
4. Architecture: **arm64**.
5. Create function.
6. **Code** tab → *Runtime settings* → Edit → Handler: `bootstrap` → Save.
7. **Configuration → General**: Memory 128 MB, Timeout 10 s.
8. **Configuration → Environment variables**:
   | Key | Value |
   |---|---|
   | `PEOPLE` | comma-separated names, e.g. `ME` (default `ME`) |
   | `ME_SECRET` | that person's key, sent by their phone shortcut in `X-API-Key` |
   | `ME_MONGODB_URI` | their MongoDB connection string |
   | `ME_MONGODB_DB` | `ledger` |
   | `<NAME>_SECRET`, `<NAME>_MONGODB_URI`, `<NAME>_MONGODB_DB` | the same three for each other name in `PEOPLE` (leave `<NAME>_SECRET` unset to switch them off) |

   Every stored message carries `user` (the lower-case name, from the key). Replaces `INGEST_SECRET`, `MONGODB_URI` and `MONGODB_DB`: update the function's variables before deploying.

   Lambda encrypts these at rest. (Use SSM SecureString later if you want rotation.)
9. **Configuration → Function URL** → Create → Auth type **NONE** → Save.
   Copy the URL (`https://<id>.lambda-url.<region>.on.aws/`).
   If you get a 403 later, open Configuration → Permissions → *Resource-based
   policy* and confirm both `lambda:InvokeFunctionUrl` and `lambda:InvokeFunction`
   statements for principal `*` exist.
10. **Configuration → Monitoring**: Log retention is set on the CloudWatch log
    group `/aws/lambda/ledger-api` (Edit retention → 14 days).

## 3. First upload by hand (to test)

From `ledger-api/`:

```
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap ./cmd/api
go install github.com/aws/aws-lambda-go/cmd/build-lambda-zip@latest
build-lambda-zip -o function.zip bootstrap
```

Console → Code tab → *Upload from* → .zip file → `function.zip`.

> On Windows do not use Explorer or `Compress-Archive`: they drop the executable
> bit and Lambda fails with `Runtime.InvalidEntrypoint`. `build-lambda-zip` (or
> `zip` on Linux/macOS/CI) is correct.

Test:

```
curl -X POST https://<id>.lambda-url.<region>.on.aws/raw-transactions \
  -H "X-API-Key: <ME_SECRET>" -H "Content-Type: application/json" \
  -d '{"message":"test"}'
```

Expect `201 {"id":"..."}`. Then change the phone shortcut URL to this Function URL.

## 4. GitHub Actions → Lambda

### One-time AWS setup (OIDC, no long-lived keys)

1. IAM → Identity providers → Add provider → OpenID Connect:
   provider URL `https://token.actions.githubusercontent.com`, audience `sts.amazonaws.com`.
2. IAM → Roles → Create role → Web identity → that provider, audience
   `sts.amazonaws.com`, then edit the trust policy so only your repo can assume it:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {"Federated": "arn:aws:iam::<ACCOUNT_ID>:oidc-provider/token.actions.githubusercontent.com"},
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {"token.actions.githubusercontent.com:aud": "sts.amazonaws.com"},
      "StringLike":   {"token.actions.githubusercontent.com:sub": "repo:<GITHUB_USER>/<REPO>:ref:refs/heads/main"}
    }
  }]
}
```

3. Attach this permission policy to the role:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["lambda:UpdateFunctionCode", "lambda:GetFunction"],
    "Resource": "arn:aws:lambda:<REGION>:<ACCOUNT_ID>:function:ledger-api"
  }]
}
```

4. GitHub repo → Settings → Secrets and variables → Actions → add secret
   `AWS_ROLE_ARN` (the role's ARN) and variable `AWS_REGION`.

### Workflow

Save as `.github/workflows/deploy-ledger-api.yml` (adjust `working-directory` if
`ledger-api` is not at the repo root):

```yaml
name: deploy-ledger-api
on:
  push:
    branches: [main]
    paths: ["ledger-api/**"]
permissions:
  id-token: write
  contents: read
defaults:
  run:
    working-directory: ledger-api
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: ledger-api/go.mod
      - run: go vet ./... && go test ./...
      - run: CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap ./cmd/api
      - run: zip function.zip bootstrap
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: ${{ vars.AWS_REGION }}
      - run: aws lambda update-function-code --function-name ledger-api --zip-file fileb://function.zip --architectures arm64
```

The workflow only replaces code. Environment variables, the Function URL and the
log retention stay as configured in the console.

## Troubleshooting

| Symptom | Cause |
|---|---|
| `Runtime.InvalidEntrypoint` / `Runtime.ExitError` at start | zip made without exec bit, binary not named `bootstrap`, or built for amd64 while the function is arm64 |
| 502 / timeout, log says `mongo ping` | Atlas allowlist missing `0.0.0.0/0`, or wrong URI |
| 401 | `X-API-Key` doesn't equal any `<NAME>_SECRET` |
| 403 from the URL itself | Function URL resource policy missing (see step 9) |
| `missing env var ...` in logs | environment variable not set on the function |

## Notes

- `ledger-backend` (parsing + AI) is a separate app and is not deployed by this
  guide; it currently runs locally and polls the same Mongo database.
- Never commit `.env`; Lambda gets its values from the function's environment.
