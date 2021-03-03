#!/usr/bin/env bash

package="${1}"
version="${2:-0.0.1}"

if [[ -z "${package}" ]]; then
  echo "usage: $0 <package-name>"
  exit 1
fi

package_split=(${package//\// })
package_name=${package_split[-1]}

platforms=( \
  "linux/amd64" \
  "windows/amd64" \
)

mkdir -p bin
mkdir -p "build/${version}"

for platform in "${platforms[@]}"
do
  echo "${platform}"

  platform_split=(${platform//\// })
  GOOS=${platform_split[0]}
  GOARCH=${platform_split[1]}
  folder_name="bin/${GOOS}/${GOARCH}/${version}"

  mkdir -p "${folder_name}" 

  output_name="${package_name}"

  if [ $GOOS = "windows" ]; then
    output_name+='.exe'
  fi

  env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="-X 'github.com/StarForger/neb-mc-rcon/cmd.BuildVersion=${version}'" \
    -a -o "${folder_name}/${output_name}" $package

  if [ $? -ne 0 ]; then
    echo 'An error has occurred! Aborting the script execution...'
    exit 1
  fi

  tar_name="${output_name}-${GOOS}-${GOARCH}.tgz"

  tar -C "${folder_name}/" -czf "build/${version}/${tar_name}" ${output_name}
done
