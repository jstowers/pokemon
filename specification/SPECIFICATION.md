# Pokemon API Coding Challenge

Wednesday, April 29, 2026

## Objective

I have Pokemon data stored in `data/pokemons.json`. Your mission is to create a PostgreSQL database and expose the database to a API.

You need to:
- Review and analyze the Pokemon data
    - How many pokemons are there?
    - What are the key attributes for each pokemon?
- Design the PostgreSQL database to store information for the Pokemon data:
    - What is the best way to store the data?
    - What is the most efficient way to access the data?
- Load the database with the data
- Implement the API Interface with the User Stories described below.

## User Stories

The API interface requires the following endpoints:

1. GET a pokemon by id

1. GET a list of pokemons, use pagination

1. GET a pokemon by name

1. GET a list of pokemons with different filters

1. GET a list of pokemons by type

1. POST add a pokemon as a favorite

1. DELETE remove a pokemon as a favorite

1. GET a list of favorite pokemons

## Technology Stack

1. Use Go as the programming language.  Use idiomatic best practices for Go in terms of organizing and structuring the code, implementing the API interface, and implementing unit tests.  Review and analyze [Effective Go](https://go.dev/doc/effective_go) for guidance.

1. Create a Swagger interface to manually test the endpoints.

1. Create a local PostgreSQL database.  Assume one user for now, but leave open the possibility to scale to multiple users.

1. The PostgreSQL database will run locally, but leave open the possibility to scale to a cloud-based SQL database like IBM Cloud, AWS, or Google Cloud.

1. Create a clear, concise README.md file with instructions on how to seed the database, start the API, and manually test the endpoints.

1. Write clear and specific unit tests to test the core functionality and each endpoint.