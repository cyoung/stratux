ifeq ($(statuxBuild),)
stratuxBuild=`git log -n 1 --pretty=%H`
buildtime=`date +%s`
endif

www=/opt/stratux/www

all:
	mkdir -p $(www)
	cp -r css $(www)/
	cp -r js $(www)/
	cp -r img $(www)/
	cp -r maui $(www)/
	cp -r plates $(www)/
	cp index.html $(www)/
	cp alert.wav $(www)/
	cp stratux.appcache $(www)/
	# Mark the manifest with the git hash.
	echo "# build time: " ${buildtime} >>$(www)/stratux.appcache
	echo "# Stratux build: " ${stratuxBuild} >>$(www)/stratux.appcache
