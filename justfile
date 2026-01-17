VERSION := "1.6.0"
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


build: update-version
  docker build --platform linux/amd64 -t juliangpurse/gopherflow:{{VERSION}} -f Dockerfile.multistage . ; docker push juliangpurse/gopherflow:{{VERSION}}

git:
    git tag v{{VERSION}}
    git push origin v{{VERSION}}

push:
  docker push juliangpurse/gopherflow:{{VERSION}}

# may be required
# go install gotest.tools/gotestsum@latest
integration-test:
     gotestsum --format short-verbose ./test/integration/...
