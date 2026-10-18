#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"
tag=${1:-}
[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Usage: scripts/release.sh vX.Y.Z' >&2; exit 2; }
[[ -z $(git status --porcelain) ]] || { echo dirty_worktree >&2; exit 2; }
git fetch origin main
[[ $(git rev-parse HEAD) == $(git rev-parse origin/main) ]] || { echo not_origin_main >&2; exit 2; }
remote=$(git remote get-url origin)
[[ "$remote" == "git@github.com:fgpaz/mi-telegram-cli.git" || "$remote" == "https://github.com/fgpaz/mi-telegram-cli.git" ]] || { echo remote_mismatch >&2; exit 2; }
[[ -z $(git tag --list "$tag") ]] || { echo tag_exists >&2; exit 2; }
if git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null 2>&1; then
  echo tag_exists >&2
  exit 2
fi

version=${tag#v}
commit=$(git rev-parse HEAD)
build_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)
stage=$(mktemp -d "${TMPDIR:-/tmp}/mi-telegram-cli-release.XXXXXX")
trap 'rm -rf "$stage"' EXIT

GOWORK=off GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -mod=readonly -trimpath -buildvcs=false \
  -ldflags "-X mi-telegram-cli/internal/management.buildVersion=$version -X mi-telegram-cli/internal/management.buildCommit=$commit -X mi-telegram-cli/internal/management.buildTime=$build_time -X mi-telegram-cli/internal/management.vcsModified=false" \
  -o "$stage/mi-telegram-cli" ./cmd/mi-telegram-cli

"$stage/mi-telegram-cli" version > "$stage/version.json"
"$stage/mi-telegram-cli" --version > "$stage/version-flag.json"
"$stage/mi-telegram-cli" provider-manifest > "$stage/provider-manifest.json"
"$stage/mi-telegram-cli" doctor > "$stage/doctor.json"
cmp "$stage/version.json" "$stage/version-flag.json"
python3 - "$stage" "$version" "$commit" "$build_time" <<'PY'
import json, pathlib, re, sys
stage = pathlib.Path(sys.argv[1])
version, commit, build_time = sys.argv[2:]
read = lambda name: json.loads((stage / name).read_text())
def check_schema(schema):
    assert isinstance(schema, dict)
    allowed = {'$schema', '$ref', '$defs', 'type', 'properties', 'required', 'additionalProperties', 'items',
               'enum', 'const', 'oneOf', 'anyOf', 'allOf', 'pattern', 'patternProperties', 'format', 'description',
               'minLength', 'maxLength', 'minimum', 'maximum'}
    assert set(schema) <= allowed, f'unsupported schema keyword: {set(schema) - allowed}'
    kind = schema.get('type')
    valid_types = {'null', 'boolean', 'object', 'array', 'number', 'integer', 'string'}
    if kind is not None:
        kinds = kind if isinstance(kind, list) else [kind]
        assert kinds and all(item in valid_types for item in kinds)
    for key in ('properties', 'patternProperties', '$defs'):
        values = schema.get(key, {})
        assert isinstance(values, dict)
        for name, child in values.items():
            if key == 'patternProperties':
                re.compile(name)
            check_schema(child)
    required = schema.get('required', [])
    assert isinstance(required, list) and all(isinstance(name, str) for name in required)
    additional = schema.get('additionalProperties')
    if isinstance(additional, dict):
        check_schema(additional)
    elif additional is not None:
        assert isinstance(additional, bool)
    if 'items' in schema:
        check_schema(schema['items'])
    for key in ('oneOf', 'anyOf', 'allOf'):
        choices = schema.get(key, [])
        assert isinstance(choices, list) and choices
        for child in choices:
            check_schema(child)
    if 'pattern' in schema:
        re.compile(schema['pattern'])
v = read('version.json')
assert set(v) == {'ok', 'data'} and v['ok'] is True
assert set(v['data']) == {'version', 'commit', 'build_time', 'go', 'os', 'arch', 'vcs_modified'}
assert v['data']['version'] == version and v['data']['commit'] == commit
assert v['data']['build_time'] == build_time and v['data']['os'] == 'linux'
assert v['data']['arch'] == 'amd64' and v['data']['vcs_modified'] == 'false'
assert re.fullmatch(r'go1\.[0-9]+\.[0-9]+', v['data']['go'])
d = read('doctor.json')
assert d['ok'] is True and d['data']['status'] == 'ok'
assert d['data']['read_only'] is True and d['data']['redacted'] is True
m = read('provider-manifest.json')
assert m['format'] == 'mi-mcp-provider/v2' and m['namespace'] == 'telegram'
assert m['binary'] == 'mi-telegram-cli' and m['provider_version'] == version
assert isinstance(m['description'], str) and m['description']
assert isinstance(m['tags'], list) and all(isinstance(tag, str) for tag in m['tags'])
assert m['auth_env_refs'] == ['MI_TELEGRAM_API_ID', 'MI_TELEGRAM_API_HASH']
root_inv = m['invocation']
assert set(root_inv) == {'kind', 'command', 'fixed_args', 'input_mapping', 'output'}
assert root_inv['kind'] == 'cli' and root_inv['command'] == 'mi-telegram-cli'
assert root_inv['fixed_args'] == [] and root_inv['input_mapping'] == {'fields': {}} and root_inv['output'] == 'json'
ops = m['operations']
assert ops and len({op['name'] for op in ops}) == len(ops)
assert set(m['contracts']) == {op['name'] for op in ops}
expected_effects = {
    'version': 'read', 'doctor': 'read', 'projects bind': 'write', 'projects current': 'read',
    'projects list': 'read', 'auth status': 'read', 'messages send': 'write',
    'messages send-photo': 'write', 'messages read': 'read', 'messages wait': 'read',
    'messages press-button': 'write', 'mark-read': 'write', 'audit summary': 'read',
}
by_name = {}
for op in ops:
    assert {'name', 'description', 'effect', 'contract', 'input_schema', 'output_schema', 'invocation'} <= set(op)
    assert op['effect'] in {'read', 'write', 'destructive'}
    assert m['contracts'].get(op['name']) == op['contract']
    check_schema(op['input_schema'])
    check_schema(op['output_schema'])
    assert op['input_schema']['type'] == 'object'
    assert op['input_schema']['additionalProperties'] is False
    assert op['output_schema']['type'] == 'object'
    inv = op['invocation']
    assert set(inv) == {'kind', 'command', 'fixed_args', 'input_mapping', 'output'}
    assert inv['kind'] == 'cli' and inv['command'] == 'mi-telegram-cli'
    assert isinstance(inv['fixed_args'], list) and all(isinstance(arg, str) for arg in inv['fixed_args'])
    assert set(inv['input_mapping']) == {'fields'} and isinstance(inv['input_mapping']['fields'], dict)
    fields = inv['input_mapping']['fields']
    assert set(fields) == set(op['input_schema'].get('properties', {}))
    assert all(set(field) == {'source', 'flag'} and field['source'] == 'option' and field['flag'].startswith('--') for field in fields.values())
    assert inv['output'] == 'json'
    by_name[op['name']] = op['effect']
assert by_name == expected_effects
assert 'auth login' not in by_name and 'profiles add' not in by_name
PY

asset="mi-telegram-cli-${tag}-linux-amd64.tar.gz"
checksum="${asset}.sha256"
cp README.md "$stage/README.md"
tar -C "$stage" -czf "$stage/$asset" mi-telegram-cli provider-manifest.json README.md
(cd "$stage" && sha256sum "$asset" > "$checksum")
gh release create "$tag" "$stage/$asset" "$stage/$checksum" \
  --repo fgpaz/mi-telegram-cli \
  --target "$commit" \
  --title "mi-telegram-cli $tag" \
  --notes "Linux amd64 CLI built from origin/main $commit. Management contract: mi-mcp-provider/v2, namespace telegram."
printf 'release_url=https://github.com/fgpaz/mi-telegram-cli/releases/tag/%s\nrelease_tag=%s\ncommit=%s\nbinary_sha256=%s\nasset_sha256=%s\n' "$tag" "$tag" "$commit" "$(sha256sum "$stage/mi-telegram-cli" | awk '{print $1}')" "$(sha256sum "$stage/$asset" | awk '{print $1}')"
