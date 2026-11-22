"""T1+T5+T8: crawl route (app.go) vs template (forms/links) vs handler perm/CSRF.
Outputs findings to stdout as markdown rows. Read-only (no app changes)."""
import re, pathlib

ROOT = pathlib.Path(r"D:\project laravel\notaris")
app = (ROOT / "internal/app/app.go").read_text(encoding="utf-8")

# 1. routes registered
routes = []
for m in re.finditer(r'(?:r|p)\.(Get|Post|Handle)\("([^"]+)"(?:,\s*a\.(\w+))?', app):
    method, path, handler = m.group(1), m.group(2), m.group(3)
    routes.append((method, path, handler or "HANDLE/FILE"))

# 2. handlers defined + permission used inside (first rbac check in function body)
handlers = {}
for m in re.finditer(r"func \(a \*App\) (\w+)\(w http\.ResponseWriter, r \*http\.Request\) \{(.*?)\n\}\n", app.read_text(encoding="utf-8") if False else "", re.S):
    pass
for fn in pathlib.Path(ROOT / "internal/app").glob("*.go"):
    if fn.name.endswith("_test.go"):
        continue
    src = fn.read_text(encoding="utf-8")
    for m in re.finditer(r"func \(a \*App\) (\w+)\(w http\.ResponseWriter, r \*http\.Request\) \{(.*?)(?=\nfunc \(a \*App\) |\Z)", src, re.S):
        name, body = m.group(1), m.group(2)
        perm = re.search(r'Perm([A-Za-z]+)', body)
        csrf_exempt = "login" in name.lower()
        handlers[name] = {"file": fn.name, "perm": ("Perm" + perm.group(1)) if perm else "NONE-FOUND", "body": body}

print(f"## Routes: {len(routes)} | Handlers: {len(handlers)}")
print("\n### Routes without handler function")
for method, path, h in routes:
    if h in ("HANDLE/FILE",):
        continue
    if h not in handlers:
        print(f"- MISSING-HANDLER {method} {path} -> a.{h}")

print("\n### Handlers without rbac/perm check (possible gap; verify manually)")
for name, h in sorted(handlers.items()):
    b = h["body"]
    if "rbac.Has" not in b and "requireStaff" not in b and "requireWeb" not in b and "RoleOf" not in b:
        print(f"- NO-PERM-CHECK a.{name} ({h['file']})")

print("\n### POST routes (CSRF: global middleware except /login)")
for method, path, h in routes:
    if method == "Post":
        print(f"- POST {path} -> {h}")

# 3. template forms vs routes
forms = {}
for tpl in (ROOT / "internal/app/templates").glob("*.gohtml"):
    src = tpl.read_text(encoding="utf-8")
    for m in re.finditer(r'<form[^>]*method="post"[^>]*action="([^"]+)"', src):
        act = m.group(1)
        act = re.sub(r"\{\{.*?\}\}", "X", act)
        forms.setdefault(act, []).append(tpl.name)

print("\n### Template POST actions with no matching route (params normalized)")
def norm(p):
    return re.sub(r"\{[^}]+\}", "X", p)
routepaths = {norm(p) for _, p, _ in routes}
for act, tpls in sorted(forms.items()):
    a = act.replace("{{.Data.", "X").replace("}}", "")
    hit = any(re.sub(r"X+", "X", a).rstrip("/") == re.sub(r"X+", "X", r).rstrip("/") or a.split("?")[0] in (r,) for r in routepaths)
    if not hit:
        # try template-param match
        t = re.sub(r"\{\{[^}]*\}\}", "X", act)
        if t not in routepaths and not any(t.startswith(r.split("X")[0]) for r in routepaths if "X" in r):
            print(f"- NO-ROUTE action={act} in {','.join(set(tpls))}")
print("\nDONE")
