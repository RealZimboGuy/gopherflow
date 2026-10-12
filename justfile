VERSION := "1.9.1"
# just build git push
update-version:
    @echo "Updating version to {{VERSION}}"
    @echo "Updating version to {{VERSION}}"

    # Update only 'juliangpurse/gopherflow:X.Y.Z' in Dockerfile
    sed -i '' -E 's|(juliangpurse/gopherflow:)[0-9]+\.[0-9]+\.[0-9]+|\1{{VERSION}}|g' Dockerfile.multistage

    # Update only the Docker image version in README
    sed -i '' -E 's|(juliangpurse/gopherflow:)[0-9]+\.[0-9]+\.[0-9]+|\1{{VERSION}}|g' README.md

    # Update only the go-get module line in README
    sed -i '' -E 's|(github\.com/RealZimboGuy/gopherflow@)v?[0-9]+\.[0-9]+\.[0-9]+|\1v{{VERSION}}|g' README.md


# Builds amd64 and arm64 in one manifest and pushes it. buildx --push is
# required: a multi-platform image cannot be loaded into the local daemon.
# Docker's default builder cannot build multi-platform images, so this uses a
# docker-container builder, created on first use. It is not made the default.
build: update-version
  docker buildx inspect gopherflow-builder >/dev/null 2>&1 || docker buildx create --name gopherflow-builder --driver docker-container --bootstrap
  docker buildx build --builder gopherflow-builder --platform linux/amd64,linux/arm64 -t juliangpurse/gopherflow:{{VERSION}} -f Dockerfile.multistage --push .

git:
    git tag v{{VERSION}}
    git push origin v{{VERSION}}

push:
  docker push juliangpurse/gopherflow:{{VERSION}}

# may be required
# go install gotest.tools/gotestsum@latest
integration-test:
     gotestsum --format short-verbose ./test/integration/...
