#!/bin/bash

function show_help() {
	echo "Git LFS Lite"
       	echo "To select which files will be using git LFS Lite add #GitLFSLite to your .gitignore and add the files or file types you want to track after that line"	
	echo "Usage:"
	echo "gitlfslite summaries"
	echo "	create sha256 summaries to check the integrity of files later"
	echo "gitlfslite check"
	echo "	check the integrity of all files"
	echo "gitlfslite rsynclist"
	echo "	create file to sync files between cloned repos with rsync"
	echo "  sync repo files with the command"
	echo "  rsync -v -t --files-from=rsync_list.gitlfslite . [destination]"  
}

function get_lfslite_files() {
	LFSLiteComment=0 
	LFSLiteFiles=()
	while read line; do
		if echo "$LFSLiteComment" | grep -q "1"; then
			LFSLiteFiles+=$(find . -type f -iname "$line")
			LFSLiteFiles+=" " 
		fi  

		if echo "$line" | grep -q "#GitLFSLite"; then 
			LFSLiteComment=1
		fi
	done < ".gitignore"

	echo $LFSLiteFiles
}

function create_summaries() {
	echo "Create summaries" 
	FILELIST=$(get_lfslite_files)

	for FILE in $FILELIST;
	do
		SUMMARYFILE=$FILE".shasum" 		
		DATEFILE=$FILE".gitlfslite" 		

		shasum -a 256 $FILE > $SUMMARYFILE
		date -r $FILE > $DATEFILE
	done
}

function check_summaries() {
	echo "Check summaries" 
	FILELIST=$(find . -type f -iname "*.shasum")

	for FILE in $FILELIST;
	do
		RESPONSE=$(shasum -c $FILE)

		if echo "$RESPONSE" | grep -q "FAILED"; then
			echo $RESPONSE
			exit 1
		fi

		echo $RESPONSE 
	done
	
}

function create_rsync_list() {
	echo "Create RSYNC list" 
	FILELIST=$(get_lfslite_files)

	LISTNAME="rsync_list.gitlfslite" 

	if [ -f $LISTNAME ]; then
		rm $LISTNAME
	fi

	for FILE in $FILELIST;
	do
		echo "$FILE" >> $LISTNAME
	done
	
}

if [ -d ".git" ]; then
	if [ -f ".gitignore" ]; then
		COMMAND="$1"

		case "$1" in
			check) shift;		check_summaries ;;
			summaries) shift;	create_summaries ;;
			rsynclist) shift;	create_rsync_list ;;
			*)			show_help ;;
		esac

		exit 0
	else
		echo ".gitignore file not found"
		exit 1 
	fi 
else
	if [ -z "$1" ]; then
		show_help
	else
		echo "Git repo folder not found, run this script in the git repo root"
		exit 1
	fi

	exit 0
fi
