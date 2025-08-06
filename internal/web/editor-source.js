import { EditorView, basicSetup } from "codemirror"
import { EditorState } from "@codemirror/state"

const textarea = document.querySelector("#content");
const form = document.querySelector("#editForm"); // Get form by ID
const finalSaveButton = document.querySelector("#finalSaveButton");

// Create the editor state with the content from the textarea
const state = EditorState.create({
    doc: textarea.value,
    extensions: [
        basicSetup,
        // A more modern theme that integrates with Bootstrap's CSS variables
        EditorView.theme({
            "&": {
                color: "var(--bs-body-color)",
                backgroundColor: "var(--bs-body-bg)",
                height: "100%", // Fill the parent container
            },
            // Apply the font directly to the scrollable content area
            ".cm-scroller": {
                fontFamily: "'JetBrains Mono', monospace",
                overflow: "auto" // Ensure scroller is always present
            },
            ".cm-content": {
                caretColor: "var(--bs-body-color)"
            },
            "&.cm-focused": {
                outline: "0",
            },
            ".cm-gutters": {
                backgroundColor: "transparent", // Gutter now blends with the editor background
                color: "var(--bs-secondary-color)",
                border: "none" // No right-side border on the gutter
            },
            // Style the cursor to be visible in both light and dark modes
            ".cm-cursor, .cm-dropCursor": {
                borderLeftColor: "var(--bs-body-color)"
            },
            // Remove the active line highlight
            ".cm-activeLineGutter": {
                backgroundColor: "transparent"
            },
            ".cm-activeLine": {
                backgroundColor: "transparent"
            },
            "& .cm-selectionBackground, ::selection": {
                backgroundColor: "rgba(var(--bs-primary-rgb), 0.2) !important",
            },
        })
    ]
});

// The parent for the editor is now the div wrapping the textarea
const editorParent = textarea.parentElement;

const view = new EditorView({
    state,
    parent: editorParent
});

// Hide the original textarea so the user only sees the editor
textarea.style.display = "none";

// Listen for the final save button click in the modal
if (finalSaveButton && form) {
    finalSaveButton.addEventListener('click', () => {
        // 1. Update the hidden textarea with the editor's content
        textarea.value = view.state.doc.toString();

        // 2. Get the comment from the modal's input
        const comment = document.querySelector("#modalComment").value;

        // 3. Create a hidden input for the comment and add it to the form
        let commentInput = form.querySelector('input[name="comment"]');
        if (!commentInput) {
            commentInput = document.createElement('input');
            commentInput.type = 'hidden';
            commentInput.name = 'comment';
            form.appendChild(commentInput);
        }
        commentInput.value = comment;

        // 4. Submit the form
        form.submit();
    });
}
