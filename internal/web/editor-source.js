import { EditorView, basicSetup } from "codemirror"
import { EditorState } from "@codemirror/state"

// --- Preview Logic ---

// Debounce function to limit how often the preview is updated
let debounceTimer;
function debounce(func, timeout = 300){
  return (...args) => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => { func.apply(this, args); }, timeout);
  };
}

// Function to fetch and render the preview
async function updatePreview(docText) {
    const previewPane = document.querySelector("#preview-content");
    if (!previewPane) return;

    try {
        const response = await fetch('/_preview', {
            method: 'POST',
            headers: {
                'Content-Type': 'text/plain; charset=utf-8',
            },
            body: docText
        });
        if (response.ok) {
            const html = await response.text();
            previewPane.innerHTML = html;
        } else {
            previewPane.innerHTML = '<div class="alert alert-danger">Error loading preview.</div>';
        }
    } catch (error) {
        console.error('Preview Error:', error);
        previewPane.innerHTML = '<div class="alert alert-danger">Could not connect to server for preview.</div>';
    }
}

const debouncedUpdatePreview = debounce(updatePreview, 250);

// --- Upload Logic ---
function createUploadButton(view) {
    const button = document.createElement('button');
    // Use standard button classes and adjust margin
    button.className = 'btn btn-outline-secondary me-2'; 
    button.type = 'button'; // Prevent form submission
    button.innerHTML = `
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-upload" viewBox="0 0 16 16">
          <path d="M.5 9.9a.5.5 0 0 1 .5.5v2.5a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-2.5a.5.5 0 0 1 1 0v2.5a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2v-2.5a.5.5 0 0 1 .5-.5"/>
          <path d="M7.646 1.146a.5.5 0 0 1 .708 0l3 3a.5.5 0 0 1-.708.708L8.5 2.707V11.5a.5.5 0 0 1-1 0V2.707L5.354 4.854a.5.5 0 1 1-.708-.708z"/>
        </svg>
        Upload File`;
    button.onclick = (e) => {
        e.preventDefault();
        const input = document.createElement('input');
        input.type = 'file';
        input.onchange = async () => {
            if (input.files.length === 0) return;
            const file = input.files[0];
            const formData = new FormData();
            formData.append('file', file);

            try {
                const response = await fetch('/upload', {
                    method: 'POST',
                    body: formData
                });
                if (response.ok) {
                    const data = await response.json();
                    // Insert the org-mode link at the current cursor position
                    const link = `[[${data.url}]]`;
                    view.dispatch({
                        changes: {from: view.state.selection.main.head, insert: link}
                    });
                } else {
                    alert('File upload failed.');
                }
            } catch (error) {
                console.error('Upload error:', error);
                alert('An error occurred during upload.');
            }
        };
        input.click();
    };
    return button;
}

// --- Editor Setup ---

const textarea = document.querySelector("#content");
const form = document.querySelector("form"); // Find the first form on the page
const finalSaveButton = document.querySelector("#finalSaveButton");

const state = EditorState.create({
    doc: textarea.value,
    extensions: [
        basicSetup,
        EditorView.theme({
            "&": {
                color: "var(--bs-body-color)",
                backgroundColor: "var(--bs-body-bg)",
                height: "100%",
            },
            ".cm-scroller": {
                fontFamily: "'JetBrains Mono', monospace",
                overflow: "auto"
            },
            ".cm-content": {
                caretColor: "var(--bs-body-color)"
            },
            "&.cm-focused": {
                outline: "0",
            },
            ".cm-gutters": {
                backgroundColor: "transparent",
                color: "var(--bs-secondary-color)",
                border: "none"
            },
            ".cm-cursor, .cm-dropCursor": {
                borderLeftColor: "var(--bs-body-color)"
            },
            ".cm-activeLineGutter": {
                backgroundColor: "transparent"
            },
            ".cm-activeLine": {
                backgroundColor: "transparent"
            },
            "& .cm-selectionBackground, ::selection": {
                backgroundColor: "rgba(var(--bs-primary-rgb), 0.2) !important",
            },
        }),
        EditorView.updateListener.of(update => {
            if (update.docChanged) {
                debouncedUpdatePreview(update.state.doc.toString());
            }
        })
    ]
});

const editorParent = textarea.parentElement;
const view = new EditorView({
    state,
    parent: editorParent
});

// --- Form Submission Logic ---

if (finalSaveButton && form) {
    finalSaveButton.addEventListener('click', () => {
        textarea.value = view.state.doc.toString();
        const comment = document.querySelector("#modalComment").value;
        let commentInput = form.querySelector('input[name="comment"]');
        if (!commentInput) {
            commentInput = document.createElement('input');
            commentInput.type = 'hidden';
            commentInput.name = 'comment';
            form.appendChild(commentInput);
        }
        commentInput.value = comment;
        form.submit();
    });
}

// --- Add Upload Button to the Form Header ---
if (form) {
    const header = form.querySelector("h1");
    if(header) {
        const button = createUploadButton(view);
        header.parentElement.querySelector("div").prepend(button);
    }
}

// --- Initial Load ---

// Trigger an initial preview render when the page loads
updatePreview(textarea.value);
