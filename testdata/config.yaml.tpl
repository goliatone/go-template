name: {{ name }}
version: {{ version }}
environment: {{ environment }}
features:
{% for feature in features %}  - {{ feature }}
{% endfor %}