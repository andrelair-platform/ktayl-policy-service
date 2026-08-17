import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    {type: 'doc', id: 'intro', label: 'Overview'},
    {
      type: 'category',
      label: 'Data Model',
      collapsed: false,
      items: [
        'data-model/mcd',
        'data-model/mld',
        'data-model/mpd',
      ],
    },
    {
      type: 'category',
      label: 'Architecture Decisions',
      collapsed: false,
      items: [
        'adr/ADR-001-database-layer',
      ],
    },
  ],
};

export default sidebars;
