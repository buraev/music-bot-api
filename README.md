# Music Bot API

This is a simple REST API for managing links and tags related to music content. It uses SQLite as a database backend.

---

## Getting Started

### Requirements

- Go 1.18+
- SQLite3

### Configuration

Create a config file or set environment variables as needed for:

- `DatabasePath` — path to SQLite database file
- `Port` — port to run the server on (default: 8080)

---

## API Endpoints

### Tags

- **GET /tags**

  Returns a list of all tags sorted by name.

  **Response:**

  ```json
  [
    { "id": "uuid", "name": "tagName" },
    ...
  ]
  ```

- **POST /tags**

  Creates a new tag.

  **Request body:**

  ```json
  { "name": "tagName" }
  ```

  **Response:**

  ```json
  { "id": "uuid", "name": "tagName" }
  ```

- **PUT /tags**

  Updates an existing tag.

  **Request body:**

  ```json
  { "id": "uuid", "name": "newTagName" }
  ```

- **DELETE /tags?id={tagID}**

  Deletes the tag with the given ID and removes all related link associations.

---

### Links

- **GET /links**

  Returns all links with their associated tags.

  Optional query parameter:

  - `tags` — comma-separated list of tag names to filter links by (links must have all tags).

  **Example:** `/links?tags=rock,pop`

  **Response:**

  ```json
  [
    {
      "id": "uuid",
      "url": "http://example.com",
      "tags": [
        { "id": "uuid", "name": "rock" },
        { "id": "uuid", "name": "pop" }
      ]
    },
    ...
  ]
  ```

- **POST /links**

  Creates or updates a link and assigns tags to it.

  **Request body:**

  ```json
  {
    "url": "http://example.com",
    "tags": [
      { "id": "uuid" },
      ...
    ]
  }
  ```

  **Response:**

  ```json
  {
    "id": "uuid",
    "url": "http://example.com",
    "tags": [
      { "id": "uuid", "name": "tagName" },
      ...
    ]
  }
  ```

- **DELETE /links?id={linkID}**

  Deletes the link with the given ID and removes all related tag associations.

---

## CORS Support

This API supports CORS with permissive settings to allow cross-origin requests from any domain.

---

## License

MIT License.

