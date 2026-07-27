const dialog = document.querySelector('[data-search-dialog]');
const input = document.querySelector('[data-search-input]');
const items = [...document.querySelectorAll('[data-search-item]')];

const openSearch = () => {
  if (!(dialog instanceof HTMLDialogElement)) return;
  dialog.showModal();
  if (input instanceof HTMLInputElement) setTimeout(() => input.focus(), 20);
};

document.querySelector('[data-open-search]')?.addEventListener('click', openSearch);
document.addEventListener('keydown', event => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault();
    openSearch();
  }
});

input?.addEventListener('input', () => {
  const query = input instanceof HTMLInputElement ? input.value.trim().toLowerCase() : '';
  items.forEach(item => {
    if (!(item instanceof HTMLElement)) return;
    item.hidden = Boolean(query) && !item.dataset.searchItem?.includes(query);
  });
});

document.querySelectorAll('pre').forEach(block => {
  const button = document.createElement('button');
  button.className = 'copy-button';
  button.type = 'button';
  button.textContent = '复制';
  button.addEventListener('click', async () => {
    const code = block.querySelector('code')?.textContent || '';
    await navigator.clipboard.writeText(code.trim());
    button.textContent = '已复制';
    setTimeout(() => (button.textContent = '复制'), 1200);
  });
  block.append(button);
});
