Pod::Spec.new do |spec|
  spec.name         = 'Gfaf'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/fafereum/go-fafereum'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS fafereum Client'
  spec.source       = { :git => 'https://github.com/fafereum/go-fafereum.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gfaf.framework'

	spec.prepare_command = <<-CMD
    curl https://gfafstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gfaf.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
