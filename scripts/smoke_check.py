from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = [
    "docker-compose.yml",
    ".env.example",
    "backend/cmd/server/main.go",
    "backend/migrations/001_init.sql",
    "ai-service/app/main.py",
    "web/app/page.tsx",
    "web/components/dashboard.tsx",
]

missing = [item for item in REQUIRED if not (ROOT / item).is_file()]
if missing:
    print("Missing required files:", ", ".join(missing))
    sys.exit(1)

backend_text = "\n".join(
    path.read_text(encoding="utf-8", errors="ignore")
    for path in (ROOT / "backend").rglob("*.go")
)
for forbidden in ("createOrder", "placeOrder", "/orders", "auto_execution"):
    if forbidden.lower() in backend_text.lower():
        print(f"Forbidden execution capability detected: {forbidden}")
        sys.exit(1)

print("Structure OK. No automatic order endpoint detected.")
