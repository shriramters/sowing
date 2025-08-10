/*!
 * Color mode toggler for Bootstrap's docs (https://getbootstrap.com/)
 * Copyright 2011-2024 The Bootstrap Authors
 * Licensed under the Creative Commons Attribution 3.0 Unported License.
 */

(() => {
    'use strict'

    const getStoredTheme = () => localStorage.getItem('theme')
    const setStoredTheme = theme => localStorage.setItem('theme', theme)

    const getPreferredTheme = () => {
        const storedTheme = getStoredTheme()
        if (storedTheme) {
            return storedTheme
        }

        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    }

    const setTheme = theme => {
        if (theme === 'auto' && window.matchMedia('(prefers-color-scheme: dark)').matches) {
            document.documentElement.setAttribute('data-bs-theme', 'dark')
        } else {
            document.documentElement.setAttribute('data-bs-theme', theme)
        }
    }

    setTheme(getPreferredTheme())

    window.addEventListener('DOMContentLoaded', () => {
        document.querySelectorAll('[data-bs-theme-value]')
            .forEach(toggle => {
                toggle.addEventListener('click', () => {
                    const theme = toggle.getAttribute('data-bs-theme-value')
                    setStoredTheme(theme)
                    setTheme(theme)
                })
            })
    })
})()

// Sidebar toggle script
document.addEventListener('DOMContentLoaded', function () {
    const sidebarToggle = document.getElementById('sidebar-toggle');
    const sidebar = document.getElementById('sidebar');
    const mainWrapper = document.getElementById('main-wrapper');

    // Function to set sidebar state
    const setSidebarState = (collapsed) => {
        if (collapsed) {
            sidebar.classList.add('collapsed');
            mainWrapper.classList.add('sidebar-collapsed');
        } else {
            sidebar.classList.remove('collapsed');
            mainWrapper.classList.remove('sidebar-collapsed');
        }
    };

    // Check for saved sidebar state in localStorage
    const isSidebarCollapsed = localStorage.getItem('sidebarCollapsed') === 'true';
    setSidebarState(isSidebarCollapsed);

    // Remove the no-transition class after the initial state is set
    setTimeout(() => {
        sidebar.classList.remove('no-transition');
    }, 100); // A short delay to ensure the initial state is applied before enabling transitions

    if (sidebarToggle && sidebar && mainWrapper) {
        sidebarToggle.addEventListener('click', function () {
            const shouldBeCollapsed = !sidebar.classList.contains('collapsed');
            setSidebarState(shouldBeCollapsed);
            localStorage.setItem('sidebarCollapsed', shouldBeCollapsed);
        });
    }
});
