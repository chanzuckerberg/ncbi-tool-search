* Unpolished script-style utility and example functions used in NCBI NT/NR research and experimentation to match sequences by accession number to smaller files.

* Components:
  * Sync service: https://github.com/chanzuckerberg/ncbi-tool-sync
  * Server service: https://github.com/chanzuckerberg/ncbi-tool-server
  * Command line client: https://github.com/chanzuckerberg/ncbi-tool-cliclient
  * Search tool utility: https://github.com/chanzuckerberg/ncbi-tool-search

- Folder structure for search utility functions:
  - accession_extraction.go
    - Utility functions for extracting accession numbers from files in remote directories.
  - main.go
    - Barebones entry point.
  - prefix_extraction.go
    - Functions for simply getting lists of all the prefixes found in the files.
  - prefix_search.go
    - Main flow used for going from accession numbers to hits/matches found in smaller files in target search directories.
  - range_reduction.go
    - Functions for formatting accession numbers and reformatting point values into ranges.
  - util.go
    - Utility functions for error handling and such.