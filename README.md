# Raven

Local bug-report portal (product name **Raven**): Gin + SQLite API, Vue + Tailwind + Shadcn-style + Vazirmatn WebUI, installable PWA (Workbox), browser Web Push.

## Run

```powershell
# ensure nginx gateway network exists
docker network create pc-armin-local 2>$null

cd C:\Users\armin\GitHub\raven
docker compose up -d
```

Open: **https://pc-armin:8443/bugs/** (self-signed TLS on the nginx gateway; host port **8443** because SoftEther VPN already uses 443). HTTP on port 80 still works for other apps; use HTTPS here so closed-tab Web Push can run.

### Trust the self-signed cert (once)

Gateway certs live under `armin-command-center/.armin/nginx-docker/certs/`. Generate/regenerate with `certs/generate-pc-armin-cert.ps1`, then:

```powershell
Import-Certificate -FilePath "C:\Users\armin\GitHub\armin-command-center\.armin\nginx-docker\certs\pc-armin.crt" -CertStoreLocation Cert:\LocalMachine\Root
```

(Run elevated if LocalMachine import is denied.) After trust, restart the browser and open `https://pc-armin:8443/bugs/`.

After sign-in, use **Enable notifications** in the banner and choose **Allow** in the browser prompt. On plain HTTP, in-app notifications (badge / toast polling) remain reliable.

## Users (password `123`)

| Username | Role |
|----------|------|
| Admin | admin (can delete bugs) |
| Developer | developer |
| Manager | manager |
| Tester1, Tester2 | tester |

## Workflow statuses

| Status | Meaning (badge) |
|--------|-----------------|
| reported | Waiting for &lt;DEV&gt; to fix it |
| fixed | Waiting for &lt;TESTER&gt; to approve it |
| waiting_manager | Waiting for &lt;MANAGER&gt; to finalize it |
| done | The task was closed in &lt;Shamsi date&gt; |

Flow: **Tester** reports → Dev marks fixed → **that same tester** approve/reject loop → Manager finalize/reject. After Dev marks fixed, only the reporting tester is notified and can approve/reject (admin can override). Tester or manager reject returns the bug to `reported` for the developer; after the next fix it goes back to the reporting tester (`fixed`), never straight to the manager.

**Manager** reports → Dev marks fixed → **Manager finalize** (skips Tester). Status badges: Waiting for &lt;DEV&gt; to fix it → Waiting for &lt;MANAGER&gt; to finalize it. Manager reject returns to Dev; the next fix goes back to Manager again.

Accept / reject actions are on the **bug detail** page. The notifications list only opens the bug (and mark-read).

## Web Push env

Set on the API service (replace the `REPLACE_WITH_GENERATED_VAPID_*` placeholders in `docker-compose.yml`):

- `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` — generate with:

```powershell
cd api
@'
package main
import (
  "fmt"
  webpush "github.com/SherClockHolmes/webpush-go"
)
func main() {
  priv, pub, err := webpush.GenerateVAPIDKeys()
  if err != nil { panic(err) }
  fmt.Println("public:", pub)
  fmt.Println("private:", priv)
}
'@ | Set-Content tmp_vapid.go
go run .\tmp_vapid.go
Remove-Item .\tmp_vapid.go
```

- `VAPID_SUBJECT` — e.g. `mailto:bugs@raven.local`
- `PUBLIC_BASE_URL` — e.g. `https://pc-armin:8443/bugs` (used in notification click URLs)
