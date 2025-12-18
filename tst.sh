#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${DEVBOX_SHELL_ENABLED:-}" ]]; then
  echo "Please run inside devbox: devbox shell"
  exit 1
fi

ROOT_DIR=$(pwd)
TMPDIR=$(mktemp -d)
cd "$TMPDIR"

if [[ "${USE_LOCAL:-0}" == "1" ]]; then
  echo "Building local plugins..."
  (cd "$ROOT_DIR" && make build)
  cp "$ROOT_DIR/build/"*.wasm .
else
  echo "Downloading formatters from latest release..."
  RELEASE_URL="https://github.com/mridang/dprint-goat/releases/latest/download"
  curl -sL "$RELEASE_URL/gofmt.wasm" -o gofmt.wasm
  curl -sL "$RELEASE_URL/shfmt.wasm" -o shfmt.wasm
  curl -sL "$RELEASE_URL/tffmt.wasm" -o tffmt.wasm
  curl -sL "$RELEASE_URL/cuefmt.wasm" -o cuefmt.wasm
  curl -sL "$RELEASE_URL/protofmt.wasm" -o protofmt.wasm
fi

write_config() {
  local include_proto="$1"
  local includes=( "**/*.go" "**/*.sh" "**/*.bash" "**/*.tf" "**/*.cue" )
  local plugins=( "./gofmt.wasm" "./shfmt.wasm" "./tffmt.wasm" "./cuefmt.wasm" )

  if [[ "$include_proto" == "1" ]]; then
    includes+=( "**/*.proto" )
    plugins+=( "./protofmt.wasm" )
  fi

  {
    echo '{'
    echo '  "includes": ['
    for i in "${!includes[@]}"; do
      local comma=","
      if [[ "$i" -eq "$((${#includes[@]} - 1))" ]]; then
        comma=""
      fi
      printf '    "%s"%s\n' "${includes[$i]}" "$comma"
    done
    echo '  ],'
    echo '  "plugins": ['
    for i in "${!plugins[@]}"; do
      local comma=","
      if [[ "$i" -eq "$((${#plugins[@]} - 1))" ]]; then
        comma=""
      fi
      printf '    "%s"%s\n' "${plugins[$i]}" "$comma"
    done
    echo '  ]'
    echo '}'
  } > dprint.json
}

if [[ "${SKIP_PROTOFMT:-0}" == "1" ]]; then
  write_config 0
else
  write_config 1
fi

cat > main.go << 'EOF'
package main
import "fmt"
func main(  ){
        fmt.Println(  "hello"  )
    }
EOF

cat > script.sh << 'EOF'
#!/bin/bash
if [ -f "test" ];then
    echo "exists"
        fi
for i in 1 2 3;do
echo $i
done
EOF

cat > main.tf << 'EOF'
resource "aws_instance" "example"{
ami="ami-12345"
instance_type="t2.micro"
    tags={
Name="test"
    }
}
EOF

cat > service.proto << 'EOF'
syntax="proto3";
package api;
message User{
string name=1;
 int32 age=2;
}
service UserService{
rpc GetUser(User)returns(User);
}
EOF

cat > schema.cue << 'EOF'
package main
import "list"
foo:  {
  bar: "baz"
  num: 1
}
EOF

cp main.go main.go.bak
cp script.sh script.sh.bak
cp main.tf main.tf.bak
cp schema.cue schema.cue.bak
cp service.proto service.proto.bak

if [[ "${SKIP_PROTOFMT:-0}" == "1" ]]; then
  rm -f service.proto
fi

if ! dprint fmt --log-level=warn; then
  echo "dprint failed; retrying without protofmt..."
  write_config 0
  rm -f service.proto
  dprint fmt --log-level=warn
fi

echo "=== main.go ==="
diff -y --width=150 main.go.bak main.go || true
echo -e "\n=== script.sh ==="
diff -y --width=150 script.sh.bak script.sh || true
echo -e "\n=== main.tf ==="
diff -y --width=150 main.tf.bak main.tf || true
if [[ -f service.proto ]]; then
  echo -e "\n=== service.proto ==="
  diff -y --width=150 service.proto.bak service.proto || true
else
  echo -e "\n=== service.proto ==="
  echo "skipped (protofmt disabled or failed)."
fi
echo -e "\n=== schema.cue ==="
diff -y --width=150 schema.cue.bak schema.cue || true

echo -e "\nTest directory: $TMPDIR"
if [[ "${USE_LOCAL:-0}" == "1" ]]; then
  echo "Local plugins built from: $ROOT_DIR"
fi
