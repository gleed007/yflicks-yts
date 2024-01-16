#!/bin/bash
release_dir="dist"
release_version=$1
release_name="yflicks-yts-$release_version"
release_path="$release_dir/$release_name"

git-chglog --next-tag $release_version --output CHANGELOG.md 
git add CHANGELOG.md
git commit -m "chore(release): v$release_version"
git tag -sa -m "yflicks-yts-$release_version" v$release_version 

rm -rf $release_dir
source_files=($(ls --ignore={.git,.gitignore,Makefile,githooks,scripts}))
mkdir -p $release_path

for source_file in ${source_files[@]}; do 
  cp -r $source_file $release_path
done

cd $release_dir && echo $release_name.tar.gz
tar -czvf $release_name.tar.gz $release_name

echo && echo $release_name.zip
zip -r $release_name.zip $release_name
rm -rf $release_name