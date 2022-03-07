#!/bin/bash

echo "Git LFS Lite" 

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
	FILELIST=$(get_lfslite_files)

	for FILE in $FILELIST;
	do
		SUMMARYFILE=$FILE".shasum" 		

		RESPONSE=$(shasum -c $SUMMARYFILE)

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

#create_summaries
#check_summaries
create_rsync_list
#get_lfslite_files
