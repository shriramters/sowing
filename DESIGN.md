# Project Specification: SOWING

Simple Org-mode WIki eNGine

## 1. Project Overview

**SOWING** is a self-contained, web-based wiki engine written in Go. It organizes content into top-level **Silos** (e.g., "Infrastructure", "Back Office"). Authenticated users can create and edit pages within these silos, while anonymous users can view them. The system supports a hierarchical page structure within each Silo and manages a full revision history. The entire application should be a single Go binary.

## 2. Core Architecture

* **Language:** Go
* **Backend:** Use the standard `net/http` library for the web server.
* **Parsing:** Use the `go-org` library to parse Org mode content into HTML.
* **Database:** Use an embedded database, specifically **SQLite**, to store all data. The application should interact with it via the standard `database/sql` package and a driver like `mattn/go-sqlite3`.
* **Templating:** Use Go's `html/template` package for server-side rendering of HTML pages.

## 3. Database Schema

The schema is updated to include `silos`, `users`, modular `identities`, and hierarchical pages.

### `silos` Table (New)

Stores metadata for each top-level content area.

| Column      | Type      | Constraints                    | Description                                          |
| :---------- | :-------- | :----------------------------- | :--------------------------------------------------- |
| `id`        | `INTEGER` | `PRIMARY KEY`, `AUTOINCREMENT` | Unique identifier for the silo.                      |
| `slug`      | `TEXT`    | `UNIQUE`, `NOT NULL`           | A URL-friendly identifier (e.g., "infra").           |
| `name`      | `TEXT`    | `NOT NULL`                     | The human-readable name (e.g., "Infrastructure").    |
| `archived_at` | `TIMESTAMP` | `NULL`                         | Timestamp for soft-deletion. `NULL` if not archived. |

### `pages` Table

Each page now belongs to a Silo and can have a parent page within that silo.

| Column                | Type      | Constraints                         | Description                                                              |
| :-------------------- | :-------- | :---------------------------------- | :----------------------------------------------------------------------- |
| `id`                  | `INTEGER` | `PRIMARY KEY`, `AUTOINCREMENT`      | Unique identifier for the page.                                          |
| `silo_id`             | `INTEGER` | `NOT NULL`, `FOREIGN KEY(silos.id)` | The ID of the silo this page belongs to.                                 |
| `parent_id`           | `INTEGER` | `NULL`, `FOREIGN KEY(pages.id)`     | The ID of the parent page. `NULL` for top-level pages within a silo.     |
| `slug`                | `TEXT`    | `NOT NULL`                          | A URL-friendly identifier for the page, unique within its parent. **Immutable after creation.** |
| `title`               | `TEXT`    | `NOT NULL`                          | The human-readable page title.                                           |
| `current_revision_id` | `INTEGER` | `NOT NULL`                          | A foreign key pointing to the active revision in the `revisions` table. |
| `archived_at`         | `TIMESTAMP` | `NULL`                              | Timestamp for soft-deletion. `NULL` if not archived.                     |

*\*Note: A composite unique constraint should be enforced on `(silo_id, parent_id, slug)`. For top-level pages where `parent_id` is `NULL`, a partial index (in PostgreSQL) or a unique index with a `WHERE parent_id IS NULL` clause (in SQLite) should be used to ensure uniqueness.*

### `revisions` Table

Revisions are linked to an author from the `users` table.

| Column       | Type        | Constraints                               | Description                                                      |
| :----------- | :---------- | :---------------------------------------- | :--------------------------------------------------------------- |
| `id`         | `INTEGER`   | `PRIMARY KEY`, `AUTOINCREMENT`            | Unique identifier for the revision.                              |
| `page_id`    | `INTEGER`   | `NOT NULL`                                | Foreign key linking to the `pages` table.                        |
| `content`    | `TEXT`      | `NOT NULL`                                | The full, raw Org mode text for this specific revision.          |
| `author_id`  | `INTEGER`   | `NOT NULL`                                | Foreign key linking to the user who saved this revision.         |
| `comment`    | `TEXT`      |                                           | An "edit summary" message for this revision.                     |
| `created_at` | `TIMESTAMP` | `NOT NULL`, `DEFAULT CURRENT_TIMESTAMP`   | The timestamp when the revision was saved.                       |

### `users` Table

Stores user profile information.

| Column         | Type      | Constraints                    | Description                               |
| :------------- | :-------- | :----------------------------- | :---------------------------------------- |
| `id`           | `INTEGER` | `PRIMARY KEY`, `AUTOINCREMENT` | Unique identifier for the user.           |
| `username`     | `TEXT`    | `UNIQUE`, `NOT NULL`           | A unique, stable username.                |
| `display_name` | `TEXT`    | `NOT NULL`                     | The user's full name for display purposes. |

### `identities` Table

This table provides modular authentication, linking login methods to users.

| Column             | Type      | Constraints                    | Description                                                  |
| :----------------- | :-------- | :----------------------------- | :----------------------------------------------------------- |
| `id`               | `INTEGER` | `PRIMARY KEY`, `AUTOINCREMENT` | Unique identifier for the identity.                          |
| `user_id`          | `INTEGER` | `NOT NULL`                     | Foreign key linking to the `users` table.                    |
| `provider`         | `TEXT`    | `NOT NULL`                     | The auth provider (e.g., 'local', 'oidc', 'saml').           |
| `provider_user_id` | `TEXT`    | `NOT NULL`                     | The user's unique ID from the provider (e.g., email or sub). |
| `password_hash`    | `TEXT`    |                                | Securely hashed password (only for 'local' provider).        |

## 4. Web UI & Template Specifications

The UI will be implemented using a server-side rendering (SSR) approach with standard Go libraries and minimal frontend tooling to align with the project's self-contained philosophy.

*   **Templating:** Go's `html/template` package will be used for all server-side rendering. Templates will be stored in the `web/templates/` directory.
*   **Styling:** The **Bootstrap** CSS framework will be used for a clean and modern look. It will be included via a CDN in the main layout file to avoid local asset management.
*   **Frontend JavaScript:** JavaScript usage will be minimal. The **CodeMirror** library will be integrated to provide a rich text editor for Org mode content. Its assets will be stored in the `web/static/` directory.

### Homepage (`/`)

* Displays a list of all available **Silos**. Each silo name links to its root page (e.g., `/infra/wiki`).

### Master Layout (`layout.html`)

* **Navigation Bar:** Includes a link back to the Silo list (`/`). If inside a silo, the silo's name is prominently displayed. It also adapts to show Login/Logout buttons based on the user's session.

### View Page Template (`view.html`)

* **Breadcrumbs:** At the top, display a breadcrumb trail: `Home > {Silo Name} > {Parent Page} > ... > {Current Page}`.
* **Child Pages:** At the bottom, list all immediate child pages.
* **Actions:** "Edit this page" (links to `.../path/to/page/edit`), "View History" (links to `.../path/to/page/history`), and "Archive" buttons. The "Edit" and "Archive" buttons are only visible to logged-in users.

### Edit Page Template (`edit.html`)

* **Editor Library:** Integrate a library like **CodeMirror** to provide a rich editing experience.
* **Form:** The form submits to the current URL (e.g., `POST` to `.../path/to/page/edit`).
* **Inputs:**
    * A dropdown to select the **Silo** (disabled if editing an existing page).
    * A dropdown to select the **Parent Page** within the chosen silo. This can be changed to move the page.
    * A text input for the `slug` (disabled if editing an existing page).
    * A `<textarea>` for the Org mode `content`, enhanced by the CodeMirror editor.
    * A text input for the `comment`.

## 5. Feature & Endpoint Specifications

### System Administration

* **Initial Setup:** The application binary will support a command-line interface for administrative tasks.
    * `sowing admin create-user --username <name> --display-name <display>`: Creates a new user and prompts for a password.
    * `sowing admin create-silo --name <name> --slug <slug>`: Creates a new silo.

### Authentication

* The system uses secure sessions for login state. Endpoints for `/login` and `/logout` will be provided for the initial `'local'` provider.

### View a Page

* **Route:** `GET /{silo_slug}/wiki/*` (e.g., `/infra/wiki/deployment-guide`)
* **Logic:**
    1.  Find the silo by its `silo_slug`.
    2.  Resolve the hierarchical page path within that silo.
    3.  Render the page with breadcrumbs and child pages.

### Create a New Page

* **Route:** `GET /{silo_slug}/_new`
* **Authorization:** Requires login.
* **Logic:** Renders a creation form similar to the edit page template, but with the `silo` and `slug` fields enabled.
* **Route:** `POST /{silo_slug}/_new`
* **Authorization:** Requires login.
* **Logic:**
    1.  Parse and validate form data.
    2.  Create the new page and its initial revision in a transaction.
    3.  Redirect to the new page's view URL.

### Edit a Page

* **Route:** `GET /{silo_slug}/wiki/*/edit`
* **Authorization:** Requires login.
* **Logic:**
    1.  Resolve the silo and page path.
    2.  Render the edit form.

### Save a New Revision

* **Route:** `POST /{silo_slug}/wiki/*/edit`
* **Authorization:** Requires login.
* **Logic:**
    1.  Resolve the silo and page path.
    2.  Parse form data.
    3.  Get `author_id` from the user's session.
    4.  In a transaction, create/update the page and insert the new revision.
    5.  Redirect to the page's view URL.

### View Page History

* **Route:** `GET /{silo_slug}/wiki/*/history`
* **Logic:** Resolves the silo and page path before showing history.

### Compare Revisions

* **Route:** `GET /{silo_slug}/wiki/*/diff`
* **Logic:** Resolves the silo and page path before showing the diff.

### Archiving and Restoration

* **Archive Page:** `POST /{silo_slug}/wiki/*/archive`
* **Restore Page:** `POST /{silo_slug}/wiki/*/restore`
* **Archive Silo:** `POST /{silo_slug}/archive`
* **Restore Silo:** `POST /{silo_slug}/restore`
* **Authorization:** Requires login.
* **Logic:** These endpoints will perform soft deletes by setting the `archived_at` timestamp.

### Exporting

* **Silo Export:** `GET /export/dump/{silo_slug}`
* **Logic:** Exports all pages for a *specific silo* as a zip archive.
* **Full System Export:** `GET /export/dump/all`
* **Authorization:** Requires admin privileges.
* **Logic:** Exports the entire SQLite database file.
