# sowing

Simple Org-mode WIki eNGine

**WORK IN PROGRESS, DO NOT USE UNTIL A TAGGED RELEASE IS AVAILABLE**

<img width="1921" height="1080" alt="image" src="https://github.com/user-attachments/assets/495277e8-769e-4983-a919-05f06b5654e2" />

## Features

*   **Org-mode Content:** Pages are written in Org mode, a powerful and flexible plain-text format.
*   **Hierarchical Pages:** Organize content in a tree-like structure within top-level "Silos".
*   **Revision History:** Every change to a page is saved, with the ability to view history and compare revisions.
*   **User Authentication:** A simple authentication system allows users to create and edit pages.
*   **Single Binary:** The entire application is a single Go binary, making deployment easy.

## Tech Stack

*   **Backend:** Go (using the standard `net/http` library)
*   **Database:** SQLite (embedded)
*   **Frontend:** Server-side rendered HTML with Bootstrap and CodeMirror

## Getting Started

1.  **Build the application:**

    Use `make build` to compile the application and prepare web assets. This command will also install Node.js dependencies (like Bootstrap) if they are not already present.

    ```bash
    make build
    ```

2.  **Create an initial user and silo:**

    ```bash
    ./sowing admin create-user --username <name> --display-name <display> --password <password>
    ./sowing admin create-silo --name <name> --slug <slug>
    ```

3.  **Run the server:**

    ```bash
    ./sowing
    ```

## License

This project is licensed under the AGPL-3.0 License. See the `LICENSE` file for details.
