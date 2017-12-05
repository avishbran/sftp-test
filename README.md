# sftp-test
SFTP test application

Usage of sftp-test:
  -I uint
        Interval length between requests [seconds] (default 1)
  -P string
        Password (default "Admin")
  -T uint
        Total requests to send (default 1)
  -U string
        Username (default "Admin")
  -h string
        SFTP server hostname or IP address
  -local-dir string
        Local directory to save files (use with -read)
  -p string
        SFTP server port (default "22")
  -r string
        Regex for file search
  -read
        Read and save the remote file(s)                        Default: current working directory, or a temp directory in case of a failure to get the WD
  -remote-dir string
        Remote directory to search (default "/")
  -rename
        Enable renaming matched files
  -rename-suffix string
        Rename suffix (use with -rename)
