GOPKG ?=	ultre.me/speechotron
DOCKER_IMAGE ?=	ultreme/speechotron
GOBINS ?=	.
NPM_PACKAGES ?=	.

all: test install

-include rules.mk
