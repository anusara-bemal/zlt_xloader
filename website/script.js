/* zlt_xloader field manual — vanilla interactions, no dependencies. */
(() => {
  'use strict';

  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  /* Command catalogue: name, access path, summary, syntax, doc anchor.
     Source of truth: repository README and the Cobra command definitions. */
  const commands = [
    { name: 'scan', group: 'api', summary: 'Reachability, TCP port scan and firmware fingerprinting.', syntax: './zlt_xloader --target 192.168.1.1 scan', doc: 'commands-recon.html#scan' },
    { name: 'status', group: 'api', summary: 'Device, SIM, lock, signal and NVRAM status via shell.', syntax: './zlt_xloader status', doc: 'commands-recon.html#status' },
    { name: 'version', group: 'api', summary: 'Version, commit and build date of the binary.', syntax: './zlt_xloader version', doc: 'commands-recon.html#version' },
    { name: 'api-exploit', group: 'api', summary: 'Login, concurrent goformId sweep and bounded fuzzer.', syntax: './zlt_xloader --dry-run=false api-exploit --include-builtin', doc: 'commands-advanced.html#api-exploit' },
    { name: 'sms', group: 'api', summary: 'List stored messages or send an SMS through the router.', syntax: './zlt_xloader --dry-run=false sms list', doc: 'commands-advanced.html#messaging-and-power' },
    { name: 'ussd', group: 'api', summary: 'Send a USSD code and read the response.', syntax: './zlt_xloader --dry-run=false ussd --code "*#123#"', doc: 'commands-advanced.html#messaging-and-power' },

    { name: 'telnet', group: 'shell', summary: 'Try the known telnet/ADB payload chain and verify port 23.', syntax: './zlt_xloader --dry-run=false telnet', doc: 'commands-access.html#telnet' },
    { name: 'shell', group: 'shell', summary: 'Interactive root shell over telnet or SSH, or one command.', syntax: './zlt_xloader shell', doc: 'commands-access.html#shell' },
    { name: 'persist', group: 'shell', summary: 'Start telnet and SSH at every boot; backs up /etc/rc.', syntax: './zlt_xloader --dry-run=false --yes persist', doc: 'commands-access.html#persist' },
    { name: 'download', group: 'shell', summary: 'Export a file from the router via a local HTTP endpoint.', syntax: './zlt_xloader --dry-run=false --yes download --remote <path> --local <file>', doc: 'commands-access.html#download' },
    { name: 'unlock', group: 'shell', summary: 'Clear carrier PLMN, band and SIM locks with an NVRAM backup.', syntax: './zlt_xloader --dry-run=false --yes unlock', doc: 'commands-unlock.html#unlock' },
    { name: 'unhide', group: 'shell', summary: 'Reveal UI pages and flags hidden by the carrier firmware.', syntax: './zlt_xloader --dry-run=false --yes unhide', doc: 'commands-unlock.html#unhide' },
    { name: 'optimize', group: 'shell', summary: 'Show RF diagnostics and apply DNS, cap and QoS settings.', syntax: './zlt_xloader --dry-run=false --yes optimize', doc: 'commands-unlock.html#optimize' },
    { name: 'power', group: 'shell', summary: 'Reboot or power off the device; confirmation required.', syntax: './zlt_xloader --dry-run=false --yes power reboot', doc: 'commands-advanced.html#messaging-and-power' },

    { name: 'tr069-exploit', group: 'tr069', summary: 'Local ACS simulator that answers Inform and pushes RPCs.', syntax: './zlt_xloader tr069-exploit --rpc set-params --param <name> --value <url>', doc: 'commands-advanced.html#tr069-exploit' },

    { name: 'uart-at', group: 'uart', summary: '3.3V TTL serial interface with scripted AT send/expect.', syntax: './zlt_xloader uart-at --port COM3 --script', doc: 'commands-advanced.html#uart-at' },
    { name: 'config-decrypt', group: 'uart', summary: 'Offline config.bin container, entropy and key analysis.', syntax: './zlt_xloader config-decrypt --file ./config.bin --top 10', doc: 'commands-advanced.html#config-decrypt' },
  ];

  const index = document.getElementById('command-index');
  const searchInput = document.getElementById('command-search');
  const count = document.getElementById('command-count');
  let activeFilter = 'all';
  let query = '';

  function render() {
    if (!index) return;
    const matches = commands.filter((command) => {
      const groupOk = activeFilter === 'all' || command.group === activeFilter;
      const text = `${command.name} ${command.summary} ${command.syntax}`.toLowerCase();
      return groupOk && text.includes(query);
    });

    index.replaceChildren();
    if (!matches.length) {
      const empty = document.createElement('div');
      empty.className = 'cmd-empty';
      empty.textContent = `No entries for “${query}”. Try another name or keyword.`;
      index.append(empty);
    } else {
      const fragment = document.createDocumentFragment();
      matches.forEach((command) => {
        const row = document.createElement('div');
        row.className = 'cmd-row';

        const link = document.createElement('a');
        link.href = `docs/${command.doc}`;
        link.textContent = command.name;

        const summary = document.createElement('p');
        summary.textContent = command.summary;

        const syntax = document.createElement('code');
        syntax.textContent = command.syntax;

        row.append(link, summary, syntax);
        fragment.append(row);
      });
      index.append(fragment);
    }

    if (count) count.textContent = `${matches.length} ${matches.length === 1 ? 'entry' : 'entries'}`;
  }

  document.querySelectorAll('[data-filter]').forEach((button) => {
    button.addEventListener('click', () => {
      activeFilter = button.dataset.filter;
      document.querySelectorAll('[data-filter]').forEach((item) => {
        const active = item === button;
        item.classList.toggle('active', active);
        item.setAttribute('aria-pressed', String(active));
      });
      render();
    });
  });

  searchInput?.addEventListener('input', () => {
    query = searchInput.value.trim().toLowerCase();
    render();
  });

  render();

  /* Panels: setup tabs and workflow tabs share one small helper. */
  function wireTabs(attribute, prefix) {
    const buttons = [...document.querySelectorAll(`[data-${attribute}]`)];
    buttons.forEach((button) => {
      button.addEventListener('click', () => {
        const value = button.dataset[attribute];
        buttons.forEach((item) => {
          const active = item === button;
          item.classList.toggle('active', active);
          item.setAttribute('aria-selected', String(active));
        });
        document.querySelectorAll('[role="tabpanel"]').forEach((panel) => {
          if (prefix && !panel.id.startsWith(prefix)) return;
          const active = panel.id === (prefix ? `${prefix}${value}` : value);
          panel.classList.toggle('active', active);
          panel.hidden = !active;
        });
      });
    });
  }

  wireTabs('setup', 'setup-');
  wireTabs('work', 'work-');

  /* Copy buttons */
  document.querySelectorAll('[data-copy-text], .copy').forEach((button) => {
    button.addEventListener('click', async () => {
      let text = button.dataset.copyText;
      if (!text) {
        const block = button.closest('.code');
        text = block?.querySelector('pre')?.textContent?.trim() || '';
      }
      if (!text) return;
      try {
        await navigator.clipboard.writeText(text);
      } catch {
        const area = document.createElement('textarea');
        area.value = text;
        area.style.position = 'fixed';
        area.style.opacity = '0';
        document.body.append(area);
        area.select();
        document.execCommand('copy');
        area.remove();
      }
      if (button.textContent?.trim()) {
        const original = button.textContent;
        button.textContent = 'Copied';
        window.setTimeout(() => { button.textContent = original; }, 1200);
      }
    });
  });

  /* Mobile navigation */
  const menuButton = document.getElementById('menu-btn');
  const mobileNav = document.getElementById('mobile-nav');
  menuButton?.addEventListener('click', () => {
    const open = menuButton.getAttribute('aria-expanded') === 'true';
    menuButton.setAttribute('aria-expanded', String(!open));
    mobileNav?.classList.toggle('open', !open);
  });
  mobileNav?.querySelectorAll('a').forEach((link) => {
    link.addEventListener('click', () => {
      menuButton?.setAttribute('aria-expanded', 'false');
      mobileNav.classList.remove('open');
    });
  });

  /* Scroll-spy for hash links in the primary nav only */
  const navLinks = [...document.querySelectorAll('.nav a')]
    .filter((link) => (link.getAttribute('href') || '').startsWith('#'));
  const sections = navLinks
    .map((link) => document.querySelector(link.getAttribute('href')))
    .filter(Boolean);

  if ('IntersectionObserver' in window && sections.length) {
    const observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) return;
        navLinks.forEach((link) => {
          link.classList.toggle('active', link.getAttribute('href') === `#${entry.target.id}`);
        });
      });
    }, { rootMargin: '-20% 0px -70% 0px', threshold: 0 });
    sections.forEach((section) => observer.observe(section));
  }

  if (reduceMotion) document.documentElement.style.scrollBehavior = 'auto';
})();
