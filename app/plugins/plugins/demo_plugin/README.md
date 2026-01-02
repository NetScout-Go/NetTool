# Demo Plugin

A demonstration plugin that showcases all NetTool plugin features for developers.

## Features Demonstrated

### 1. Parameter Types

| Type | Example | Description |
|------|---------|-------------|
| `select` | Demo Mode dropdown | Dropdown with predefined options |
| `string` | Text Input | Free-form text entry |
| `number` | Number Input | Numeric value with min/max |
| `range` | Range Slider | Slider with step increments |
| `boolean` | Enable Features | Toggle switch |

### 2. Display Types (`_display`)

**Metrics** - Key-value pairs with icons and colors
```json
{
  "type": "metrics",
  "title": "Section Title",
  "color": "cyan",
  "metrics": [
    { "label": "Label", "value": "Value", "icon": "icon-name", "color": "green" }
  ],
  "extra": { "progress": 75 }
}
```

**Table** - Tabular data with typed columns
```json
{
  "type": "table",
  "title": "Table Title",
  "columns": [
    { "key": "name", "label": "Name", "type": "text" },
    { "key": "status", "label": "Status", "type": "status" }
  ],
  "data": [
    { "name": "Item", "status": "up" }
  ]
}
```

### 3. Column Types

- `text` - Plain text
- `number` - Numeric value
- `status` - Shows up/down with colored indicator
- `bytes` - Auto-formats to KB/MB/GB
- `progress` - Progress bar with percentage

### 4. Colors

`primary`, `cyan`, `green`, `orange`, `purple`, `red`, `yellow`, `blue`

### 5. Error Handling

Select "Error Handling" demo mode to see structured errors:
- `errorCode` - Machine-readable code
- `error` - User-friendly message
- `errorDetails` - Technical details

### 6. Warnings

Select "Warnings Demo" to see non-critical warnings displayed in a yellow banner.

### 7. Dependencies

The `requires` field in plugin.json lists system dependencies. NetTool checks and reports their status.

## How to Use

1. Run NetTool
2. Navigate to the Demo Plugin
3. Try different "Demo Mode" options
4. Adjust parameters to see how they're passed
5. Click "Show Raw" to see the underlying JSON

## For Plugin Developers

Use this plugin as a reference when building your own plugins. The key concepts:

1. **plugin.json** - Define your plugin metadata and parameters
2. **plugin.go** - Implement `--definition` and `--execute=<json>` handlers
3. **_display** - Return formatted sections for rich UI rendering
4. Return raw data in `data` field for advanced users
