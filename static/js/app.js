// Initialize SortableJS for drag-and-drop reordering

document.addEventListener('DOMContentLoaded', function() {
    initializeSortable();
});

// Re-initialize after htmx swaps
document.addEventListener('htmx:afterSwap', function() {
    initializeSortable();
});

function initializeSortable() {
    // Projects list on home page
    const projectsList = document.getElementById('projects-list');
    if (projectsList && !projectsList.sortable) {
        projectsList.sortable = new Sortable(projectsList, {
            handle: '.drag-handle',
            animation: 150,
            ghostClass: 'sortable-ghost',
            dragClass: 'sortable-drag',
            onEnd: function(evt) {
                const ids = Array.from(projectsList.querySelectorAll('.project-card'))
                    .map(card => parseInt(card.dataset.id));

                fetch('/api/projects/reorder', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ ids: ids }),
                });
            }
        });
    }

    // Tasks list on project detail page
    const tasksList = document.getElementById('tasks-list');
    if (tasksList && !tasksList.sortable) {
        const projectId = tasksList.dataset.projectId;
        tasksList.sortable = new Sortable(tasksList, {
            handle: '.drag-handle',
            animation: 150,
            ghostClass: 'sortable-ghost',
            dragClass: 'sortable-drag',
            onEnd: function(evt) {
                const ids = Array.from(tasksList.querySelectorAll('.task-item'))
                    .map(task => parseInt(task.dataset.id));

                fetch(`/api/projects/${projectId}/tasks/reorder`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ ids: ids }),
                });
            }
        });
    }
}

// Form visibility functions
function showProjectForm() {
    const form = document.getElementById('new-project-form');
    if (form) {
        form.classList.remove('hidden');
        form.querySelector('input[name="name"]').focus();
    }
}

function showEditProjectForm(projectId) {
    const form = document.getElementById('edit-project-form');
    if (form) {
        form.classList.remove('hidden');
        form.querySelector('input[name="name"]').focus();
    }
}

function showTaskForm(projectId) {
    const form = document.getElementById('new-task-form');
    if (form) {
        form.classList.remove('hidden');
        form.querySelector('input[name="description"]').focus();
    }
}

function showEditTaskForm(taskId, projectId) {
    // For inline editing, we could fetch and show a form
    // For now, redirect to a modal or inline form
    const taskElement = document.getElementById(`task-${taskId}`);
    if (taskElement) {
        // Simple inline edit - toggle a form
        const existingForm = taskElement.querySelector('.task-form');
        if (existingForm) {
            existingForm.classList.toggle('hidden');
            return;
        }
    }
}

function hideForm(button) {
    const formContainer = button.closest('.form-container');
    if (formContainer) {
        formContainer.classList.add('hidden');
        const form = formContainer.querySelector('form');
        if (form) {
            form.reset();
        }
    }
}

// Toggle target date visibility based on project type
function toggleTargetDate(select) {
    const targetDateGroup = document.getElementById('target-date-group');
    const targetDateInput = document.getElementById('project-target-date');

    if (select.value === 'category') {
        if (targetDateGroup) {
            targetDateGroup.style.opacity = '0.5';
        }
        if (targetDateInput) {
            targetDateInput.value = '';
            targetDateInput.disabled = true;
        }
    } else {
        if (targetDateGroup) {
            targetDateGroup.style.opacity = '1';
        }
        if (targetDateInput) {
            targetDateInput.disabled = false;
        }
    }
}

// Handle delete project redirect
document.addEventListener('htmx:beforeSwap', function(event) {
    if (event.detail.xhr.getResponseHeader('HX-Redirect')) {
        window.location.href = event.detail.xhr.getResponseHeader('HX-Redirect');
    }
});
