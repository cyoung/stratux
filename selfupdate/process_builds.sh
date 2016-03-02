#!/bin/bash

ssh -i ~/.ssh/id_rsa.updates stratux-updates@updates.stratux.me 'ls -1 queue/' | while read git_hash ; do
	echo "***** Building $git_hash. *****"
	git clone https://github.com/cyoung/stratux --recursive $git_hash
	cd $git_hash
	git reset --hard $git_hash
	cd selfupdate
	./makeupdate.sh
	cd ..
	scp -i ~/.ssh/id_rsa.updates work/update*.sh stratux-updates@updates.stratux.me:finished/
	cd ..
	ssh -i ~/.ssh/id_rsa.updates stratux-updates@updates.stratux.me "rm -f queue/${git_hash}"
done