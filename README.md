# GitLFSLite

## Overview
`gitlfslite` is a tool designed to help you manage large files in your git repository. It provides a simpler alternative to options like Git LFS or Git Annex, which can be complex or costly. This tool allows you to keep big files in sync between multiple repository clones by creating text files with metadata about the large files, which can then be tracked by git.

## Motivation
Managing software projects without git is a nightmare. While git is excellent for handling text files, it struggles with large non-text files such as videos and audio files. Existing solutions like Git LFS can be expensive and have performance issues, while Git Annex is difficult to use. I wanted a simple solution where these tools are overkill. My idea was to create a text file with file information (such as sha256 sums and modification dates) for each large file, use `.gitignore` to manage which files are not saved in the repository, and use `rsync` to keep the large files in sync across multiple repo clones.

## Features
- Generates metadata files with file information (sha256 sums, modification dates).
- Uses `.gitignore` to manage files excluded from the repository.
- Synchronizes large files using `rsync`.

## Installation
To install `gitlfslite`, clone the repository and build the Go application:
```sh
git clone https://github.com/jempe/gitlfslite.git
cd gitlfslite
go install ./cmd/glflite
```


## Usage
The `glflite` tool can perform several actions to manage your large files:

```sh
glflite -action [check|update|help] -file [file|folder] -force -quiet
```

- `-action`: Specify the action to perform. Possible values are `check`, `update`, and `help`.
- `-file`: Specify the file or folder to check or update.
- `-force`: Force the action to be performed, checking files completely to confirm if they are up to date.
- `-quiet`: Prints only the summary of the files.


## Example

To check if your files are up to date:

```sh
glflite -action check
```

To update the metadata for your files:

```sh
glflite -action update
```

To sync the files, use the `rsync` command with the list of files in the `rsync_list_glflite` file:

```sh
rsync -v -t --files-from=rsync_list_glflite . [destination]
```

## Managing Files
You need to modify the `.gitignore` file in your repository to determine which files will be managed by `glflite`. Add the files or patterns you want to exclude from the repository, and they will be handled by `glflite` instead. Only the files listed after the `#GitLFSLite` comment will be managed by `glflite`.

### Example .gitignore

```gitignore
rsync_list_glflite_local
.DS_Store
*.swp
*.mp3

#GitLFSLite
*.mp4
```

## Contributing
Feel free to fork the repository and submit pull requests. For major changes, please open an issue first to discuss what you would like to change.

## License
This project is licensed under the GPLv3 License.





