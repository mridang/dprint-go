#!/usr/bin/env bash
set -euo pipefail

TMPDIR=$(mktemp -d)
cd "$TMPDIR"

echo "Downloading formatters from latest release..."
RELEASE_URL="https://github.com/mridang/dprint-go/releases/latest/download"
curl -sL "$RELEASE_URL/gofmt.wasm" -o gofmt.wasm
curl -sL "$RELEASE_URL/shfmt.wasm" -o shfmt.wasm
curl -sL "$RELEASE_URL/tffmt.wasm" -o tffmt.wasm
curl -sL "$RELEASE_URL/protofmt.wasm" -o protofmt.wasm

cat > dprint.json << 'EOF'
{
  "includes": [
    "**/*.go",
    "**/*.sh",
    "**/*.bash",
    "**/*.tf",
    "**/*.proto"
  ],
  "plugins": [
    "./gofmt.wasm",
    "./shfmt.wasm",
    "./tffmt.wasm",
    "./protofmt.wasm"
  ]
}
EOF

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

cp main.go main.go.bak
cp script.sh script.sh.bak
cp main.tf main.tf.bak
cp service.proto service.proto.bak

dprint fmt

echo "=== main.go ==="
diff -y --width=150 main.go.bak main.go || true
echo -e "\n=== script.sh ==="
diff -y --width=150 script.sh.bak script.sh || true
echo -e "\n=== main.tf ==="
diff -y --width=150 main.tf.bak main.tf || true
echo -e "\n=== service.proto ==="
diff -y --width=150 service.proto.bak service.proto || true

echo -e "\nTest directory: $TMPDIR"
