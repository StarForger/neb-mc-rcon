#!/usr/bin/env bash

package="${1}"
version="${2:-0.0.1}"

if [[ -z "${package}" ]]; then
  echo "usage: $0 <package-name>"
  exit 1
fi

package_split=(${package//\// })
package_name=${package_split[-1]}

# platforms=("windows/amd64" "linux/amd64")
platforms=("windows/amd64")



mkdir -p bin

for platform in "${platforms[@]}"
do
  platform_split=(${platform//\// })
  GOOS=${platform_split[0]}
  GOARCH=${platform_split[1]}
  folder_name="bin/${GOOS}/${GOARCH}"

  mkdir -p "${folder_name}" 

  output_name=${package_name}

  if [ $GOOS = "windows" ]; then
    output_name+='.exe'
  fi

  env GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="-X 'github.com/StarForger/neb-rcon/cmd.BuildVersion=${version}'" \
    -a -o "${folder_name}/${output_name}" $package

  if [ $? -ne 0 ]; then
    echo 'An error has occurred! Aborting the script execution...'
    exit 1
  fi

  tar_name="${output_name}-${GOOS}-${GOARCH}.tgz"

  # tar -C "${folder_name}/" -czf "bin/${tar_name}" ${output_name}
done
