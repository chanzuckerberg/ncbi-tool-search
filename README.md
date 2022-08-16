* Unpolished script-style utility and example functions used in NCBI NT/NR research and experimentation to match sequences by accession number to smaller files.

## Archived

This repository was created for experimental purposes and is no longer maintained or used.

## Usage

- Link to Usage and Development Notes and Project Write-Up: https://czi.quip.com/rwBgAebQg2Fa

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
