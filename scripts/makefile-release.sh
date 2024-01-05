#!/bin/bash
release_name="yflicks-yts-$1"
source_files=($(ls --ignore={.git,.gitignore,Makefile,githooks,scripts}))
mkdir $release_name
for source_file in ${source_files[@]}; do 
  cp -r $source_file $release_name
done

echo $release_name.tar.gz
tar -czvf $release_name.tar.gz $release_name

echo && echo $release_name.zip
zip -r $release_name.zip $release_name
rm -rf $release_name