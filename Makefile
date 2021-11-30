bind:
	gomobile bind -o android/app/prnm.aar -target=android

docker-bind:
	docker run -v $(CURDIR):/src/ --rm -it --ulimit memlock=67108864 perunnetwork/prnm-ci:latest bash -c "go mod download golang.org/x/mobile && gomobile bind -o /src/android/app/prnm.aar -target=android"

docker-test: docker-check-privilege
	docker-compose down	# Down in case that the down was aborted last time
	@echo "This will take a while. See the log in 'full.log'."
	.scripts/docker-compose-up.sh
	docker-compose down

docker-check-privilege:
	.scripts/docker-check-privilege.sh
